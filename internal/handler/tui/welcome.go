package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type WelcomeModel struct {
	parent *AppModel
}

func NewWelcomeModel(parent *AppModel) *WelcomeModel {
	return &WelcomeModel{parent: parent}
}

func (m *WelcomeModel) Init() tea.Cmd {
	// Não há inicialização assíncrona para a tela de boas-vindas
	return nil
}

func (m *WelcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// Navegar para a tela de login
			return m, m.parent.send(showLoginMsg{})
		}
	}
	return m, nil
}

func (m *WelcomeModel) View() string {
	var b strings.Builder

	title := welcomeTitleStyle.Render("🎶 Reorder Playlist TUI 🎶")
	prompt := welcomePromptStyle.Render("Pressione Enter para fazer login com Google e começar!")

	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(prompt)
	b.WriteString("\n\n")
	b.WriteString(welcomePromptStyle.Render("(Ctrl+C ou Esc para sair)"))

	return docStyle.Render(b.String())
}
