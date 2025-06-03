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
	parent        *AppModel
	playlists     []domain.Playlist
	cursor        int
	err           error
	loading       bool
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
	m.statusMessage = ""
	m.parent.logger.Info("PlaylistsModel: Init chamado, buscando playlists…")

	return func() tea.Msg {
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
		m.lastRefresh = time.Now()
		return m, nil

	case playlistLoadErrorMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		// Ctrl+C ou Esc encerra
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

		// Se houver erro e apertar Enter → volta ao login
		if m.err != nil && msg.Type == tea.KeyEnter {
			return m, m.parent.send(showLoginMsg{})
		}

		// Ctrl+R para recarregar (cooldown 5m)
		if msg.Type == tea.KeyCtrlR {
			if m.lastRefresh.IsZero() || time.Since(m.lastRefresh) >= 5*time.Minute {
				m.parent.logger.Info("PlaylistsModel: Ctrl+R pressionado — recarregando playlists…")
				return m, m.Init()
			}
			remaining := 5*time.Minute - time.Since(m.lastRefresh)
			minutos := int(remaining.Minutes())
			segundos := int(remaining.Seconds()) % 60
			m.statusMessage = fmt.Sprintf(
				"🔄 Aguarde %02d:%02d para próximo refresh (cooldown 5m).",
				minutos, segundos,
			)
			return m, nil
		}

		// Se estiver carregando ou não tiver playlists, nada faz
		if m.loading || len(m.playlists) == 0 {
			return m, nil
		}

		// Navegação normal: índice 0 = “via URL”, índices 1..N = playlists carregadas
		switch msg.Type {
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			if m.cursor < len(m.playlists) { // máximo = len(playlists)
				m.cursor++
			}
		case tea.KeyEnter:
			if m.cursor == 0 {
				// selecionou “Reordenar via URL”
				return m, m.parent.send(showURLMsg{})
			}
			// selecionou playlist existente
			selected := m.playlists[m.cursor-1]
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
		b.WriteString("Loading playlists…\n\n")
		b.WriteString(welcomePromptStyle.Render("Pressione Ctrl+R para recarregar (cooldown 5m). Ctrl+C para sair."))
		return docStyle.Render(b.String())
	}

	if m.err != nil {
		b.WriteString(errorMessageStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
		b.WriteString(welcomePromptStyle.Render("Pressione Enter para voltar ao login."))
		b.WriteString("\n")
		b.WriteString(welcomePromptStyle.Render("Pressione Ctrl+R para recarregar. Ctrl+C para sair."))
		return docStyle.Render(b.String())
	}

	// Opção 0 = “Reordenar playlist via URL”
	if m.cursor == 0 {
		b.WriteString(selectedListItemStyle.Render("Reordenar playlist via URL"))
	} else {
		b.WriteString(listItemStyle.Render("Reordenar playlist via URL"))
	}
	b.WriteString("\n")

	// Em seguida, as playlists do usuário
	for i, p := range m.playlists {
		if m.cursor == i+1 {
			b.WriteString(selectedListItemStyle.Render(p.Title))
		} else {
			b.WriteString(listItemStyle.Render(p.Title))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(welcomePromptStyle.Render("Use ↑/↓ ou j/k para navegar, Enter para selecionar."))
	b.WriteString("\n")
	b.WriteString(welcomePromptStyle.Render("Pressione Ctrl+R para recarregar (cooldown 5m). Ctrl+C para sair."))

	if m.statusMessage != "" {
		b.WriteString("\n\n")
		b.WriteString(statusMessageStyle.Render(m.statusMessage))
	}

	return docStyle.Render(b.String())
}
