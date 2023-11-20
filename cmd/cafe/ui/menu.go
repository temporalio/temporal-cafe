package ui

import (
	"log"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/temporalio/temporal-cafe/api"
)

var (
	menuFrame = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true).
			BorderForeground(lipgloss.Color("#035afc")).
			PaddingLeft(1).
			PaddingRight(1).
			Width(60).
			Height(26)

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			MarginTop(1).
			MarginBottom(1)

	itemStyle = lipgloss.NewStyle().
			Width(28)

	focusItemStyle = itemStyle.Copy().
			Foreground(lipgloss.Color("#ee6ff8"))
)

type Menu struct {
	menu      *api.Menu
	focus     bool
	items     []menuItemDelegate
	focusItem int
}

type menuMsg struct {
	menu *api.Menu
}

type incItemCountMsg struct {
	item *api.MenuItem
}

type decItemCountMsg struct {
	item *api.MenuItem
}

type setItemCountMsg struct {
	item  *api.MenuItem
	count uint32
}

func newMenu() Menu {
	return Menu{}
}

func (m Menu) load() tea.Msg {
	return menuMsg{
		&api.Menu{
			Items: []api.MenuItem{
				{Name: "Coffee", Type: "beverage", Price: 300},
				{Name: "Latte", Type: "beverage", Price: 350},
				{Name: "Milkshake", Type: "beverage", Price: 450},
				{Name: "Bagel", Type: "food", Price: 500},
				{Name: "Sandwich", Type: "food", Price: 600},
			},
		},
	}
}

func (m Menu) Init() tea.Cmd {
	return m.load
}

func (m Menu) Update(msg tea.Msg) (Menu, tea.Cmd) {
	log.Printf("Menu: %v", msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			return m, m.focusPrevious()
		case "down":
			return m, m.focusNext()
		default:
			return m, m.updateFocused(msg)
		}
	case clickMsg:
		return m, m.updateFocused(msg)
	case menuMsg:
		m.menu = msg.menu
		for i := range m.menu.Items {
			m.items = append(m.items, newMenuItemDelegate(&m.menu.Items[i]))
		}
		m.focusItem = 0
		if m.focus {
			return m, m.items[m.focusItem].Focus()
		}

		return m, nil
	}

	return m, nil
}

func (m Menu) View() string {
	out := []string{
		titleStyle.Render("Menu"),
	}

	var section string
	for _, d := range m.items {
		if section != d.item.Type {
			section = d.item.Type
			out = append(out, sectionStyle.Render(typeHeader(d.item.Type)))
		}
		out = append(out, d.View())
	}

	return menuFrame.Render(lipgloss.JoinVertical(lipgloss.Left, out...))
}

func (m *Menu) focusPrevious() tea.Cmd {
	if m.focusItem == 0 {
		return nil
	}
	m.items[m.focusItem].Blur()
	m.focusItem -= 1
	return m.items[m.focusItem].Focus()
}

func (m *Menu) focusNext() tea.Cmd {
	if m.focusItem == len(m.items)-1 {
		return nil
	}
	m.items[m.focusItem].Blur()
	m.focusItem += 1
	return m.items[m.focusItem].Focus()
}

func (m Menu) updateFocused(msg tea.Msg) tea.Cmd {
	i, cmd := m.items[m.focusItem].Update(msg)
	m.items[m.focusItem] = i

	return cmd
}

func typeHeader(t string) string {
	switch t {
	case "beverage":
		return "Drinks"
	}

	s := strings.ToTitle(t)

	return s
}

func (m *Menu) Focus() tea.Cmd {
	m.focus = true
	if len(m.items) > 0 {
		return m.items[m.focusItem].Focus()
	}
	return nil
}

func (m *Menu) Blur() {
	m.focus = false
	if len(m.items) > 0 {
		m.items[m.focusItem].Blur()
	}
}

func (m *Menu) Reset() {
	wasFocused := m.focus
	m.Blur()
	m.focusItem = 0
	if wasFocused {
		m.Focus()
	}
}

func newMenuItemDelegate(item *api.MenuItem) menuItemDelegate {
	d := menuItemDelegate{item: item}
	d.buttons = []button{
		{label: "+", id: "inc"},
		{label: "-", id: "dec"},
		{label: "x", id: "reset"},
	}

	return d
}

type menuItemDelegate struct {
	item        *api.MenuItem
	focus       bool
	buttons     []button
	buttonFocus int
}

func (m menuItemDelegate) Init() tea.Cmd {
	return nil
}

func (m menuItemDelegate) Update(msg tea.Msg) (menuItemDelegate, tea.Cmd) {
	log.Printf("Menu Item: %v", msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left":
			return m, m.focusPreviousButton()
		case "right":
			return m, m.focusNextButton()
		case "enter", "space":
			return m, m.updateButton(msg)
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			c, err := strconv.Atoi(msg.String())
			if err != nil {
				return m, nil
			}
			return m, m.Set(uint32(c))
		}
	case clickMsg:
		switch msg.id {
		case "inc":
			return m, m.Inc
		case "dec":
			return m, m.Dec
		case "reset":
			return m, m.Set(0)
		}
	}

	return m, nil
}

func (m menuItemDelegate) View() string {
	s := itemStyle.Render
	if m.focus {
		s = focusItemStyle.Render
	}

	out := []string{s(m.item.Name)}
	for i := range m.buttons {
		out = append(out, m.buttons[i].View())
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, out...)
}

func (m *menuItemDelegate) Inc() tea.Msg {
	return incItemCountMsg{item: m.item}
}

func (m *menuItemDelegate) Dec() tea.Msg {
	return decItemCountMsg{item: m.item}
}

func (m *menuItemDelegate) Set(c uint32) tea.Cmd {
	return func() tea.Msg {
		return setItemCountMsg{item: m.item, count: c}
	}
}

func (m *menuItemDelegate) Focus() tea.Cmd {
	m.focus = true
	m.buttonFocus = 0
	return m.buttons[m.buttonFocus].Focus()
}

func (m *menuItemDelegate) Blur() {
	m.focus = false
	m.buttons[m.buttonFocus].Blur()
}

func (m *menuItemDelegate) focusNextButton() tea.Cmd {
	if m.buttonFocus == len(m.buttons)-1 {
		return nil
	}
	m.buttons[m.buttonFocus].Blur()
	m.buttonFocus += 1
	return m.buttons[m.buttonFocus].Focus()
}

func (m *menuItemDelegate) focusPreviousButton() tea.Cmd {
	if m.buttonFocus == 0 {
		return nil
	}
	m.buttons[m.buttonFocus].Blur()
	m.buttonFocus -= 1
	return m.buttons[m.buttonFocus].Focus()
}

func (m *menuItemDelegate) updateButton(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.buttons[m.buttonFocus], cmd = m.buttons[m.buttonFocus].Update(msg)
	return cmd
}
