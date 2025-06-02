package tui

import (
	"fmt"
	"strings"

	"TUI_playlist_reorder/internal/core/domain"
	tea "github.com/charmbracelet/bubbletea"
)

type playlistsLoadedMsg struct{ playlists []domain.Playlist }
type playlistLoadErrorMsg struct{ err error }

type PlaylistsModel struct {
	parent    *AppModel
	playlists []domain.Playlist
	cursor    int
	err       error
	loading   bool
}

func NewPlaylistsModel(parent *AppModel) *PlaylistsModel {
	return &PlaylistsModel{
		parent:  parent,
		loading: true,
	}
}

func (m *PlaylistsModel) Init() tea.Cmd {
	// Define que estamos iniciando carregamento de playlists
	m.loading = true
	m.err = nil
	m.playlists = nil
	m.parent.logger.Info("PlaylistsModel: Init chamado, buscando playlists…")

	return func() tea.Msg {
		m.parent.logger.Info("PlaylistsModel: Init chamado")
		playlists, err := m.parent.playlistUseCase.GetMinePlaylists(m.parent.appContext)
		if err != nil {
			m.parent.logger.Error("Falha ao obter playlists", err)
			return playlistLoadErrorMsg{err: err}
		}
		m.parent.logger.Info(fmt.Sprintf("Solicitação concluída: %d playlists.", len(playlists)))
		return playlistsLoadedMsg{playlists: playlists}
	}
}

func (m *PlaylistsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case playlistsLoadedMsg:
		m.loading = false
		m.playlists = msg.playlists
		m.cursor = 0
		if len(m.playlists) == 0 {
			m.err = fmt.Errorf("no playlists found")
		} else {
			m.err = nil
		}
		return m, nil

	case playlistLoadErrorMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		// Se há erro e o usuário apertar Enter, volta ao login
		if m.err != nil && msg.Type == tea.KeyEnter {
			return m, m.parent.send(showLoginMsg{})
		}
		// Enquanto estiver carregando ou não houver playlists, ignoramos setas/enter
		if m.loading || len(m.playlists) == 0 {
			return m, nil
		}
		switch msg.Type {
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			if m.cursor < len(m.playlists)-1 {
				m.cursor++
			}
		case tea.KeyEnter:
			selected := m.playlists[m.cursor]
			m.parent.logger.Info(fmt.Sprintf("Playlist selecionada: %s (ID: %s)", selected.Title, selected.ID))
			return m, m.parent.send(showReorderMsg{playlist: selected})
		}
	}

	return m, nil
}

func (m *PlaylistsModel) View() string {
	var b strings.Builder

	b.WriteString(listHeaderStyle.Render("Your Playlists"))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString("Loading playlists...\n")
		return docStyle.Render(b.String())
	}

	if m.err != nil {
		b.WriteString(errorMessageStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		if m.err.Error() == "no playlists found" {
			b.WriteString("\n\nIt seems you don't have any playlists.")
		} else {
			b.WriteString("\n\nPress Enter to voltar ao login.")
		}
		b.WriteString("\n\n")
		b.WriteString(welcomePromptStyle.Render("(Ctrl+C ou Esc para sair)"))
		return docStyle.Render(b.String())
	}

	// Se chegou aqui, há playlists para exibir
	for i, p := range m.playlists {
		item := p.Title
		if m.cursor == i {
			b.WriteString(selectedListItemStyle.Render(item))
		} else {
			b.WriteString(listItemStyle.Render(item))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(welcomePromptStyle.Render("Use ↑/↓ ou j/k para navegar, Enter para selecionar. Ctrl+C para sair."))

	return docStyle.Render(b.String())
}
