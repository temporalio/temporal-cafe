package ui

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
		MarginBottom(1).
		Bold(true).
		Underline(true)
)

type POS struct {
	menu   Menu
	order  Order
	status StatusBar
	focus  int
}

func NewPOS() POS {
	m := newMenu()
	m.Focus()

	return POS{
		menu:   m,
		order:  newOrder(),
		status: newStatusBar(),
	}
}

func (m POS) Init() tea.Cmd {
	return tea.Batch(m.menu.Init(), m.order.Init())
}

func (m POS) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf("POS: %v", msg)
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "tab":
			if m.focus == 0 {
				return m, m.focusOrder()
			}
			return m, m.updateFocused(msg)
		case "shift+tab":
			return m, m.focusMenu()
		default:
			return m, m.updateFocused(msg)
		}
	case clickMsg:
		return m, m.updateFocused(msg)
	case menuMsg:
		m.menu, cmd = m.menu.Update(msg)
		return m, cmd
	case statusMsg:
		m.status, cmd = m.status.Update(msg)
		return m, cmd
	case orderMsg:
		if msg.err != nil {
			return m, m.updateStatus(msg.status, msg.err)
		}
		m.menu.Reset()
		m.order.Reset()
		return m, tea.Batch(m.focusMenu(), m.updateStatus(msg.status, msg.err))
	case setItemCountMsg:
		m.order.SetItemCount(msg.item, msg.count)
	case incItemCountMsg:
		m.order.IncItemCount(msg.item)
		return m, nil
	case decItemCountMsg:
		m.order.DecItemCount(msg.item)
		return m, nil
	}

	return m, nil
}

func (m POS) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top,
			m.menu.View(),
			m.order.View(),
		),
		m.status.View(),
	)
}

func (m *POS) focusOrder() tea.Cmd {
	m.focus = 1
	m.menu.Blur()
	return m.order.Focus()
}

func (m *POS) focusMenu() tea.Cmd {
	m.focus = 0
	m.order.Blur()
	return m.menu.Focus()
}

func (m *POS) updateFocused(msg tea.Msg) tea.Cmd {
	if m.focus == 0 {
		x, cmd := m.menu.Update(msg)
		m.menu = x
		return cmd
	} else {
		x, cmd := m.order.Update(msg)
		m.order = x
		return cmd
	}
}

func (m *POS) updateStatus(status string, err error) tea.Cmd {
	return func() tea.Msg { return statusMsg{status: status, err: err} }
}
