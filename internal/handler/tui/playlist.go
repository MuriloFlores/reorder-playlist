package tui

import (
	"fmt"
	"strings"
	"time"

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

	lastRefresh   time.Time
	statusMessage string
}

func NewPlaylistsModel(parent *AppModel) *PlaylistsModel {
	return &PlaylistsModel{
		parent:        parent,
		loading:       true,
		lastRefresh:   time.Time{},
		statusMessage: "",
	}
}

func (m *PlaylistsModel) Init() tea.Cmd {
	m.loading = true
	m.err = nil
	m.playlists = nil
	m.statusMessage = "" // limpa qualquer mensagem em aberto
	m.parent.logger.Info("PlaylistsModel: Init chamado, buscando playlists…")

	return func() tea.Msg {
		playlists, err := m.parent.playlistUseCase.GetMinePlaylists(m.parent.appContext)
		if err != nil {
			m.parent.logger.Error("Falha ao obter playlists", err)
			return playlistLoadErrorMsg{err: err}
		}
		m.parent.logger.Info(fmt.Sprintf("Solicitação concluída: %d playlists encontradas.", len(playlists)))
		return playlistsLoadedMsg{playlists: playlists}
	}
}

func (m *PlaylistsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case playlistsLoadedMsg:
		// Recebe lista carregada com sucesso
		m.loading = false
		m.playlists = msg.playlists
		m.cursor = 0
		if len(m.playlists) == 0 {
			m.err = fmt.Errorf("no playlists found")
		} else {
			m.err = nil
		}

		m.lastRefresh = time.Now()
		return m, nil

	case playlistLoadErrorMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

		if m.err != nil && msg.Type == tea.KeyEnter {
			return m, m.parent.send(showLoginMsg{})
		}

		if msg.Type == tea.KeyCtrlR {
			if m.lastRefresh.IsZero() || time.Since(m.lastRefresh) >= 5*time.Minute {
				m.parent.logger.Info("PlaylistsModel: Ctrl+R pressionado — recarregando playlists…")
				return m, m.Init()
			}

			remaining := 5*time.Minute - time.Since(m.lastRefresh)
			minutos := int(remaining.Minutes())
			segundos := int(remaining.Seconds()) % 60
			m.statusMessage = fmt.Sprintf(
				"🔄 Aguarde %02d:%02d até a próxima recarga (cooldown de 5 minutos).",
				minutos, segundos,
			)
			return m, nil
		}

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
		b.WriteString("Loading playlists…\n")
		b.WriteString("\n")
		b.WriteString(welcomePromptStyle.Render("Pressione Ctrl+R para tentar recarregar. Ctrl+C para sair."))
		return docStyle.Render(b.String())
	}

	if m.err != nil {
		b.WriteString(errorMessageStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		if m.err.Error() == "no playlists found" {
			b.WriteString("\n\nIt seems you don't have any playlists.")
		} else {
			b.WriteString("\n\nPressione Enter para voltar ao login.")
		}
		b.WriteString("\n\n")
		b.WriteString(welcomePromptStyle.Render("Pressione Ctrl+R para recarregar. Ctrl+C para sair."))
		return docStyle.Render(b.String())
	}

	// Exibe as playlists carregadas
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

	// Instruções de navegação e refresh
	b.WriteString(welcomePromptStyle.Render("Use ↑/↓ para navegar, Enter para selecionar."))
	b.WriteString("\n")
	b.WriteString(welcomePromptStyle.Render("Pressione Ctrl+R para recarregar playlists (cooldown 5m). Ctrl+C para sair."))

	// Se houver mensagem de status (cooldown), exiba abaixo
	if m.statusMessage != "" {
		b.WriteString("\n\n")
		b.WriteString(statusMessageStyle.Render(m.statusMessage))
	}

	return docStyle.Render(b.String())
}
