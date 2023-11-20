package ui

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type button struct {
	label string
	id    string
	focus bool
}

type clickMsg struct {
	id string
}

var (
	labelStyle = lipgloss.NewStyle().
			Bold(true).
			MarginLeft(1).
			MarginRight(1)

	focusLabelStyle = labelStyle.Copy().
			Foreground(lipgloss.Color("#ee6ff8"))

	frameStyle = lipgloss.NewStyle()

	focusFrameStyle = lipgloss.NewStyle().
			Bold(true)
)

func (m button) Init() tea.Cmd {
	return nil
}

func (m button) Update(msg tea.Msg) (button, tea.Cmd) {
	log.Printf("Button[%s]: %v", m.id, msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "space":
			return m, func() tea.Msg { return clickMsg{id: m.id} }
		}
	}

	return m, nil
}

func (m button) View() string {
	frame := frameStyle
	label := labelStyle

	if m.focus {
		frame = focusFrameStyle
		label = focusLabelStyle
	}

	return frame.Render("[") + label.Render(m.label) + frame.Render("]")
}

func (m *button) Focus() tea.Cmd {
	m.focus = true
	return nil
}

func (m *button) Blur() {
	m.focus = false
}
