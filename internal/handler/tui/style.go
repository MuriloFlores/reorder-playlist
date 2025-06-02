package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Estilo geral de contêiner de documento (margens, padding)
	docStyle = lipgloss.NewStyle().
			Margin(1, 2)

	// Estilos para a tela de boas-vindas
	welcomeTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("62")). // Roxo
				Padding(1, 0)
	welcomePromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{
			Light: "#A49FA5",
			Dark:  "#777777",
		})

	// Estilos para listas (Playlists, opções de reorderamento, etc.)
	listHeaderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("240")). // Cinza
			MarginBottom(1).
			PaddingBottom(1)
	listItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)
	selectedListItemStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Foreground(lipgloss.Color("62")). // Roxo
				SetString("> ")

	// Estilos para mensagens de status/erro
	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{
			Light: "#04B575",
			Dark:  "#04B575",
		}) // Verde
	errorMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("9")) // Vermelho

	// Estilo para URLs (tela de login)
	urlStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")). // Azul
			Underline(true)
)
