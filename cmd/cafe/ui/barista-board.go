package ui

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/temporalio/temporal-cafe/api"
)

type baristaOrdersMsg struct {
	orders []api.BaristaOrder
}

type BaristaBoard struct {
	orders       []baristaOrder
	focusedOrder int

	err error
}

func (m BaristaBoard) Init() tea.Cmd {
	return m.fetchOrders
}

func (m BaristaBoard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf("Board: %v", msg)

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "left":
			m.PreviousOrder()
			return m, nil
		case "right":
			m.NextOrder()
			return m, nil
		default:
			if len(m.orders) > 0 {
				m.orders[m.focusedOrder], cmd = m.orders[m.focusedOrder].Update(msg)
				return m, cmd
			}
		}
	case baristaOrdersMsg:
		m.parseOrders(msg.orders)
		if len(m.orders) > 0 {
			m.orders[0].Focus()
		}
	case baristaOrderMsg:
		for i := range m.orders {
			if msg.order.ID == m.orders[i].id {
				m.orders[i], cmd = m.orders[i].Update(msg)
				return m, cmd
			}
		}
	}

	return m, nil
}

func (m BaristaBoard) View() string {
	s := lipgloss.NewStyle().Padding(1, 2, 1, 2)

	if m.err != nil {
		return fmt.Sprintf("\nWe had some trouble: %v\n\n", m.err)
	}

	var orders []string
	for _, order := range m.orders {
		orders = append(orders, order.View())
	}

	return s.Render(lipgloss.JoinHorizontal(lipgloss.Left, orders...))
}

func (m *BaristaBoard) fetchOrders() tea.Msg {
	c := &http.Client{}
	r, err := c.Get("http://localhost:8084/barista/orders")
	if err != nil {
		log.Printf("Error: %v", err)
		return statusMsg{err: err}
	}
	defer r.Body.Close()

	if r.StatusCode < 200 || r.StatusCode >= 300 {
		return statusMsg{err: fmt.Errorf("api request failed with code: %d", r.StatusCode)}
	}

	var ordersJSON []api.BaristaOrder
	err = json.NewDecoder(r.Body).Decode(&ordersJSON)
	if err != nil {
		return statusMsg{err: err}
	}

	return baristaOrdersMsg{ordersJSON}
}

func (m *BaristaBoard) parseOrders(ordersJSON []api.BaristaOrder) {
	var orders []baristaOrder

	for _, o := range ordersJSON {
		order := baristaOrder{}
		order.parseOrder(o)
		orders = append(orders, order)
	}

	m.orders = orders
}

func (m *BaristaBoard) NextOrder() {
	if m.focusedOrder < len(m.orders)-1 {
		m.orders[m.focusedOrder].Blur()
		m.focusedOrder += 1
		m.orders[m.focusedOrder].Focus()
	}
}

func (m *BaristaBoard) PreviousOrder() {
	if m.focusedOrder > 0 {
		m.orders[m.focusedOrder].Blur()
		m.focusedOrder -= 1
		m.orders[m.focusedOrder].Focus()
	}
}
