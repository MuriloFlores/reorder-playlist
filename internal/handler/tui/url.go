package tui

import (
	"fmt"
	"strings"

	"TUI_playlist_reorder/internal/core/domain"
	"TUI_playlist_reorder/internal/core/usecases"
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

type urlEnteredMsg struct {
	playlist domain.Playlist
}
type urlEnterErrorMsg struct {
	err error
}

type URLModel struct {
	parent          *AppModel
	playlistUseCase usecases.PlaylistUseCase
	
	awaitingURL bool
	newURL      string
	err         error
}

func NewURLModel(parent *AppModel) *URLModel {
	return &URLModel{
		parent:          parent,
		playlistUseCase: parent.playlistUseCase,
		awaitingURL:     true,
		newURL:          "",
		err:             nil,
	}
}

func (m *URLModel) Init() tea.Cmd {
	m.awaitingURL = true
	m.newURL = ""
	m.err = nil
	return nil
}

func (m *URLModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			// Volta para a lista de playlists
			return m, m.parent.send(showPlaylistsMsg{})
		case tea.KeyBackspace:
			if len(m.newURL) > 0 {
				m.newURL = m.newURL[:len(m.newURL)-1]
			}
			return m, nil
		case tea.KeyEnter:
			url := strings.TrimSpace(m.newURL)
			if url == "" {
				m.err = fmt.Errorf("URL não pode ficar vazia")
				return m, nil
			}
			m.err = nil
			// Dispara o fetch da playlist via URL
			return m, func() tea.Msg {
				playlist, err := m.parent.playlistUseCase.GetPlaylistByURL(context.Background(), url)
				if err != nil {
					return urlEnterErrorMsg{err: err}
				}
				return urlEnteredMsg{playlist: playlist}
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.newURL += string(msg.Runes)
			}
			return m, nil
		}

	case urlEnterErrorMsg:
		// Erro ao buscar por URL
		m.err = msg.err
		return m, nil

	case urlEnteredMsg:
		// Recebeu a playlist com sucesso, envia para reordenação
		return m, m.parent.send(showReorderMsg{playlist: msg.playlist})
	}

	return m, nil
}

func (m *URLModel) View() string {
	var b strings.Builder
	b.WriteString(listHeaderStyle.Render("Reordenar por URL"))
	b.WriteString("\n\n")
	b.WriteString("Digite a URL da playlist do YouTube e pressione Enter:\n")
	b.WriteString(listItemStyle.Render("> " + m.newURL))
	b.WriteString("\n\n")
	if m.err != nil {
		b.WriteString(errorMessageStyle.Render(fmt.Sprintf("Erro: %v", m.err)))
		b.WriteString("\n\n")
	}
	b.WriteString(welcomePromptStyle.Render("Use Backspace para apagar, Enter para confirmar, Ctrl+C para cancelar."))
	return docStyle.Render(b.String())
}
