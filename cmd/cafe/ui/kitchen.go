package ui

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/temporalio/temporal-cafe/api"
)

var (
	kitchenOrderStyle = lipgloss.NewStyle().
				Width(32).
				Border(lipgloss.NormalBorder(), true)

	kitchenFocusedOrderStyle = kitchenOrderStyle.Copy().
					BorderForeground(lipgloss.Color("#035afc"))

	kitchenListHeaderStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				MarginLeft(1)

	kitchenCursorStyle = lipgloss.NewStyle().
				MarginLeft(1).
				MarginRight(1)
	kitchenCursorFocusedStyle = kitchenCursorStyle.Copy().
					Foreground(lipgloss.Color("#035afc"))

	kitchenMarkStyle = lipgloss.NewStyle().
				MarginRight(1)

	kitchenListItemStyle          = lipgloss.NewStyle()
	kitchenListItemCompletedStyle = lipgloss.NewStyle().Strikethrough(true)
)

type kitchenOrderMsg struct {
	order api.KitchenOrder
}

type kitchenOrder struct {
	id    string
	open  bool
	focus bool

	name      string
	items     []kitchenOrderItem
	focusItem int
}

type kitchenOrderItem struct {
	name   string
	status string
}

func (i kitchenOrderItem) NextStatus() string {
	switch i.status {
	case "pending":
		return "started"
	case "started":
		return "completed"
	}

	return ""
}

func itemCursor(focus bool) string {
	if focus {
		return kitchenCursorFocusedStyle.Render("●")
	}

	return kitchenCursorStyle.Render("∙")
}

func itemMark(item kitchenOrderItem) string {
	mark := " "
	s := kitchenMarkStyle

	if item.status == "started" {
		mark = ">"
		s = kitchenMarkStyle.Copy().Foreground(lipgloss.Color("#ffbf00"))
	} else if item.status == "completed" {
		mark = "✓"
		s = kitchenMarkStyle.Copy().Foreground(lipgloss.Color("#00ff00"))
	}

	return s.Render(mark)
}

func itemName(item kitchenOrderItem) string {
	if item.status == "completed" {
		return kitchenListItemCompletedStyle.Render(item.name)
	}

	return kitchenListItemStyle.Render(item.name)
}

func boardMark(board kitchenOrder) string {
	mark := " "
	s := kitchenMarkStyle
	if !board.open {
		mark = "✓"
		s = kitchenMarkStyle.Copy().Foreground(lipgloss.Color("#00ff00"))
	}

	return s.Render(mark)
}

func (m kitchenOrder) Init() tea.Cmd {
	return nil
}

func (m *kitchenOrder) Focus() {
	m.focus = true
}

func (m *kitchenOrder) Blur() {
	m.focus = false
}

func (m *kitchenOrder) NextItem() {
	if m.focusItem < len(m.items)-1 {
		m.focusItem += 1
	}
}

func (m *kitchenOrder) PreviousItem() {
	if m.focusItem > 0 {
		m.focusItem -= 1
	}
}

func (m *kitchenOrder) parseOrder(orderJSON api.KitchenOrder) {
	m.id = orderJSON.ID
	m.name = orderJSON.Name
	m.open = orderJSON.Open

	var items []kitchenOrderItem
	for _, i := range orderJSON.Items {
		items = append(items, kitchenOrderItem{
			name:   i.Name,
			status: i.Status,
		})
	}

	m.items = items
}

func (m *kitchenOrder) updateItemStatus(line int, status string) tea.Cmd {
	if status == "" {
		return nil
	}

	return func() tea.Msg {
		c := &http.Client{}
		r, err := c.Post(
			fmt.Sprintf("http://localhost:8084/kitchen/orders/%s/%d/status", m.id, line+1),
			"text/plain",
			strings.NewReader(status),
		)
		if err != nil {
			return statusMsg{err: err}
		}
		defer r.Body.Close()

		if r.StatusCode < 200 || r.StatusCode >= 300 {
			return statusMsg{err: fmt.Errorf("api request failed with code: %d", r.StatusCode)}
		}

		var orderJSON api.KitchenOrder
		err = json.NewDecoder(r.Body).Decode(&orderJSON)
		if err != nil {
			return statusMsg{err: err}
		}

		return kitchenOrderMsg{orderJSON}
	}
}

func (m kitchenOrder) Update(msg tea.Msg) (kitchenOrder, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.open || !m.focus {
			log.Printf("focused? %v", m)
			return m, nil
		}

		switch msg.String() {
		case "enter":
			status := m.items[m.focusItem].NextStatus()
			return m, m.updateItemStatus(m.focusItem, status)
		case "up":
			m.PreviousItem()
		case "down":
			m.NextItem()
		}
	case kitchenOrderMsg:
		if msg.order.ID != m.id {
			return m, nil
		}

		m.parseOrder(msg.order)
		if m.focus && !m.open {
			m.Blur()
		}
	}

	return m, nil
}

func (m kitchenOrder) View() string {
	log.Printf("order: %v", m)
	out := []string{
		lipgloss.JoinHorizontal(lipgloss.Top, kitchenListHeaderStyle.Render(m.name), boardMark(m)),
	}

	for i, item := range m.items {
		focused := m.open && m.focus && i == m.focusItem
		out = append(out, itemCursor(focused)+itemMark(item)+itemName(item))
	}

	s := kitchenOrderStyle
	if m.focus {
		s = kitchenFocusedOrderStyle
	}

	if !m.open {
		s = s.Copy().Faint(true)
	}

	return s.Render(
		lipgloss.JoinVertical(lipgloss.Left, out...),
	)
}
