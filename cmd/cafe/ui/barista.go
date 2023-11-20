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
	baristaOrderStyle = lipgloss.NewStyle().
				Width(32).
				Border(lipgloss.NormalBorder(), true)

	baristaFocusedOrderStyle = baristaOrderStyle.Copy().
					BorderForeground(lipgloss.Color("#035afc"))

	baristaListHeaderStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				MarginLeft(1)

	baristaCursorStyle = lipgloss.NewStyle().
				MarginLeft(1).
				MarginRight(1)
	baristaCursorFocusedStyle = baristaCursorStyle.Copy().
					Foreground(lipgloss.Color("#035afc"))

	baristaMarkStyle = lipgloss.NewStyle().
				MarginRight(1)

	baristaListItemStyle          = lipgloss.NewStyle()
	baristaListItemCompletedStyle = lipgloss.NewStyle().Strikethrough(true)
)

type baristaOrderMsg struct {
	order api.BaristaOrder
}

type baristaOrder struct {
	id    string
	open  bool
	focus bool

	name      string
	items     []baristaOrderItem
	focusItem int
}

type baristaOrderItem struct {
	name   string
	status string
}

func (i baristaOrderItem) NextStatus() string {
	switch i.status {
	case "pending":
		return "started"
	case "started":
		return "completed"
	}

	return ""
}

func baristaItemCursor(focus bool) string {
	if focus {
		return baristaCursorFocusedStyle.Render("●")
	}

	return baristaCursorStyle.Render("∙")
}

func baristaItemMark(item baristaOrderItem) string {
	mark := " "
	s := baristaMarkStyle

	if item.status == "started" {
		mark = ">"
		s = baristaMarkStyle.Copy().Foreground(lipgloss.Color("#ffbf00"))
	} else if item.status == "completed" {
		mark = "✓"
		s = baristaMarkStyle.Copy().Foreground(lipgloss.Color("#00ff00"))
	}

	return s.Render(mark)
}

func baristaItemName(item baristaOrderItem) string {
	if item.status == "completed" {
		return baristaListItemCompletedStyle.Render(item.name)
	}

	return baristaListItemStyle.Render(item.name)
}

func baristaBoardMark(board baristaOrder) string {
	mark := " "
	s := baristaMarkStyle
	if !board.open {
		mark = "✓"
		s = baristaMarkStyle.Copy().Foreground(lipgloss.Color("#00ff00"))
	}

	return s.Render(mark)
}

func (m baristaOrder) Init() tea.Cmd {
	return nil
}

func (m *baristaOrder) Focus() {
	m.focus = true
}

func (m *baristaOrder) Blur() {
	m.focus = false
}

func (m *baristaOrder) NextItem() {
	if m.focusItem < len(m.items)-1 {
		m.focusItem += 1
	}
}

func (m *baristaOrder) PreviousItem() {
	if m.focusItem > 0 {
		m.focusItem -= 1
	}
}

func (m *baristaOrder) parseOrder(orderJSON api.BaristaOrder) {
	m.id = orderJSON.ID
	m.name = orderJSON.Name
	m.open = orderJSON.Open

	var items []baristaOrderItem
	for _, i := range orderJSON.Items {
		items = append(items, baristaOrderItem{
			name:   i.Name,
			status: i.Status,
		})
	}

	m.items = items
}

func (m *baristaOrder) updateItemStatus(line int, status string) tea.Cmd {
	if status == "" {
		return nil
	}

	return func() tea.Msg {
		c := &http.Client{}
		r, err := c.Post(
			fmt.Sprintf("http://localhost:8084/barista/orders/%s/%d/status", m.id, line+1),
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

		var orderJSON api.BaristaOrder
		err = json.NewDecoder(r.Body).Decode(&orderJSON)
		if err != nil {
			return statusMsg{err: err}
		}

		log.Printf("received: %v", orderJSON)

		return baristaOrderMsg{orderJSON}
	}
}

func (m baristaOrder) Update(msg tea.Msg) (baristaOrder, tea.Cmd) {
	log.Printf("Order[%s]: %v", m.id, msg)

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
	case baristaOrderMsg:
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

func (m baristaOrder) View() string {
	log.Printf("order: %v", m)
	out := []string{
		lipgloss.JoinHorizontal(lipgloss.Top, baristaListHeaderStyle.Render(m.name), baristaBoardMark(m)),
	}

	for i, item := range m.items {
		focused := m.open && m.focus && i == m.focusItem
		out = append(out, baristaItemCursor(focused)+baristaItemMark(item)+baristaItemName(item))
	}

	s := baristaOrderStyle
	if m.focus {
		s = baristaFocusedOrderStyle
	}

	if !m.open {
		s = s.Copy().Faint(true)
	}

	return s.Render(
		lipgloss.JoinVertical(lipgloss.Left, out...),
	)
}
