package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	statusResetDelay = 5 * time.Second
)

var (
	statusBarFrame = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true).
			BorderForeground(lipgloss.Color("#ffffff")).
			PaddingLeft(1).
			PaddingRight(1).
			Width(98).
			Height(1)

	faintStatusBarFrame = statusBarFrame.Copy().
				BorderForeground(lipgloss.Color("#aaaaaa"))

	errorStatusBarFrame = statusBarFrame.Copy().
				BorderForeground(lipgloss.Color("#fc0303"))
)

type statusMsg struct {
	status string
	err    error
}

type StatusBar struct {
	status string
	err    error
}

func newStatusBar() StatusBar {
	return StatusBar{}
}

func (m StatusBar) Init() tea.Cmd {
	return nil
}

func (m StatusBar) Update(msg tea.Msg) (StatusBar, tea.Cmd) {
	switch msg := msg.(type) {
	case statusMsg:
		m.err = msg.err
		m.status = msg.status

		return m, m.timeoutStatus(statusResetDelay)
	}

	return m, nil
}

func (m StatusBar) View() string {
	f := faintStatusBarFrame
	if m.status != "" {
		f = statusBarFrame
	}
	if m.err != nil {
		f = errorStatusBarFrame
	}

	return f.Render(m.status)
}

func (m StatusBar) timeoutStatus(delay time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(delay)

		return statusMsg{status: "", err: nil}
	}
}
