package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/temporalio/temporal-cafe/api"
)

var (
	orderFrame = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(lipgloss.Color("#1403fc")).
		PaddingLeft(1).
		PaddingRight(1).
		Width(36).
		Height(26)
)

type orderMsg struct {
	status string
	err    error
}

type orderFocusField int

const (
	nameField orderFocusField = iota
	emailField
	submitButton
)

type Order struct {
	order      api.Order
	name       textinput.Model
	email      textinput.Model
	submit     button
	focus      bool
	focusField orderFocusField
}

func newOrder() Order {
	name := textinput.New()
	name.Placeholder = "Name"
	email := textinput.New()
	email.Placeholder = "Email"

	order := Order{
		name:  name,
		email: email,
	}

	order.submit = button{label: "Place Order", id: "submit"}

	return order
}

func (m Order) Init() tea.Cmd {
	return nil
}

func (m Order) Update(msg tea.Msg) (Order, tea.Cmd) {
	log.Printf("Order: %v", msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			return m, m.focusPrevious()
		case "down", "tab":
			return m, m.focusNext()
		case "enter":
			if m.focusField == submitButton {
				return m, m.updateFocused(msg)
			}
			return m, m.focusNext()
		default:
			return m, m.updateFocused(msg)
		}
	case clickMsg:
		return m, m.placeOrder
	}
	return m, nil
}

func (m Order) View() string {
	out := []string{
		titleStyle.Render("Order"),
		m.name.View(),
		m.email.View(),
	}

	items := []string{}
	total := uint32(0)
	for _, item := range m.order.Items {
		if item.Count == 0 {
			continue
		}
		total += item.Price * item.Count
		items = append(items, fmt.Sprintf("%s (%s) x %d", item.Name, formatPrice(item.Price), item.Count))
	}
	items = append(items, "", fmt.Sprintf("Total: %s", formatPrice(total)))

	if len(items) > 0 {
		out = append(out, "", lipgloss.JoinVertical(lipgloss.Left, items...), "", m.submit.View())
	}

	return orderFrame.Render(
		lipgloss.JoinVertical(lipgloss.Left, out...),
	)
}

func formatPrice(p uint32) string {
	return fmt.Sprintf("$%.2f", float32(p)/100)
}

func (m *Order) updateFocused(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch m.focusField {
	case nameField:
		m.name, cmd = m.name.Update(msg)
	case emailField:
		m.email, cmd = m.email.Update(msg)
	case submitButton:
		m.submit, cmd = m.submit.Update(msg)
	}

	return cmd
}

func (m *Order) Focus() tea.Cmd {
	m.focus = true
	return m.FocusField()
}

func (m *Order) FocusField() tea.Cmd {
	switch m.focusField {
	case nameField:
		return m.name.Focus()
	case emailField:
		return m.email.Focus()
	case submitButton:
		return m.submit.Focus()
	}
	return nil
}

func (m *Order) focusNext() tea.Cmd {
	if m.focusField == submitButton {
		return nil
	}

	m.BlurField()
	m.focusField += 1
	return m.FocusField()
}

func (m *Order) focusPrevious() tea.Cmd {
	if m.focusField == nameField {
		return nil
	}

	m.BlurField()
	m.focusField -= 1
	return m.FocusField()
}

func (m *Order) Blur() {
	m.focus = false
	m.BlurField()
}

func (m *Order) BlurField() {
	switch m.focusField {
	case nameField:
		m.name.Blur()
	case emailField:
		m.email.Blur()
	case submitButton:
		m.submit.Blur()
	}
}

func (m *Order) IncItemCount(item *api.MenuItem) {
	for i := range m.order.Items {
		if item.Type == m.order.Items[i].Type && item.Name == m.order.Items[i].Name {
			m.order.Items[i].Count += 1
			return
		}
	}
	m.order.Items = append(m.order.Items, api.OrderItem{
		Type: item.Type, Name: item.Name, Price: item.Price, Count: 1,
	})
}

func (m *Order) DecItemCount(item *api.MenuItem) {
	for i := range m.order.Items {
		if item.Type == m.order.Items[i].Type && item.Name == m.order.Items[i].Name {
			if m.order.Items[i].Count > 0 {
				m.order.Items[i].Count -= 1
			}
			return
		}
	}
}

func (m *Order) SetItemCount(item *api.MenuItem, count uint32) {
	for i := range m.order.Items {
		if item.Type == m.order.Items[i].Type && item.Name == m.order.Items[i].Name {
			m.order.Items[i].Count = count
			return
		}
	}
	if count == 0 {
		return
	}

	m.order.Items = append(m.order.Items, api.OrderItem{
		Type: item.Type, Name: item.Name, Price: item.Price, Count: count,
	})
}

func (m *Order) placeOrder() tea.Msg {
	m.order.Name = m.name.Value()
	m.order.Email = m.email.Value()

	jsonInput, err := json.Marshal(m.order)
	if err != nil {
		return orderMsg{err: fmt.Errorf("unable to encode order: %w", err)}
	}

	c := &http.Client{}
	r, err := c.Post(
		"http://localhost:8084/orders",
		"application/json",
		bytes.NewReader(jsonInput),
	)
	if err != nil {
		return orderMsg{err: err}
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(r.Body)
		return orderMsg{err: fmt.Errorf("%s: %s", http.StatusText(r.StatusCode), body)}
	}

	return orderMsg{status: "Order placed"}
}

func (m *Order) Reset() {
	m.order = api.Order{}
	m.name.Reset()
	m.email.Reset()
}
