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

type playlistViaURLMsg struct {
	playlist domain.Playlist
}

type PlaylistsModel struct {
	parent        *AppModel
	playlists     []domain.Playlist
	cursor        int
	err           error
	loading       bool
	lastRefresh   time.Time
	statusMessage string

	awaitingURL bool
	newURL      string
	urlError    error
}

func NewPlaylistsModel(parent *AppModel) *PlaylistsModel {
	return &PlaylistsModel{
		parent:        parent,
		loading:       true,
		lastRefresh:   time.Time{},
		statusMessage: "",
		awaitingURL:   false,
		newURL:        "",
		urlError:      nil,
	}
}

func (m *PlaylistsModel) Init() tea.Cmd {
	// Dispara o fetch das playlists
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
		// Sempre permite Ctrl+C ou Esc para sair
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

		// Se já estivermos no modo de entrada de URL, tratamos teclas de texto
		if m.awaitingURL {
			switch msg.Type {
			case tea.KeyEnter:
				// O usuário finalizou a URL
				m.awaitingURL = false
				url := strings.TrimSpace(m.newURL)
				if url == "" {
					m.urlError = fmt.Errorf("URL não pode ficar vazia")
					m.statusMessage = "URL não pode ficar vazia. Tente novamente."
					m.newURL = ""
					return m, nil
				}

				// Exibe carregando e dispara o fetch via URL
				m.loading = true
				m.err = nil
				m.statusMessage = ""
				m.urlError = nil

				return m, func() tea.Msg {
					playlist, err := m.parent.playlistUseCase.GetPlaylistByURL(m.parent.appContext, url)
					if err != nil {
						return playlistLoadErrorMsg{err: err}
					}
					// Se for bem‐sucedido, retornamos uma mensagem especial com a playlist carregada
					return playlistViaURLMsg{playlist: playlist}
				}

			case tea.KeyBackspace:
				if len(m.newURL) > 0 {
					m.newURL = m.newURL[:len(m.newURL)-1]
				}
				return m, nil

			default:
				if msg.Type == tea.KeyRunes {
					m.newURL += string(msg.Runes)
				}
				return m, nil
			}
		}

		// Se houver erro geral e usuário apertar Enter, volta ao login
		if m.err != nil && msg.Type == tea.KeyEnter {
			return m, m.parent.send(showLoginMsg{})
		}

		// Tratamento de Ctrl+R (refresh) com cooldown de 5 minutos
		if msg.Type == tea.KeyCtrlR {
			if m.lastRefresh.IsZero() || time.Since(m.lastRefresh) >= 5*time.Minute {
				m.parent.logger.Info("PlaylistsModel: Ctrl+R pressionado — recarregando playlists…")
				return m, m.Init()
			}
			remaining := 5*time.Minute - time.Since(m.lastRefresh)
			minutos := int(remaining.Minutes())
			segundos := int(remaining.Seconds()) % 60
			m.statusMessage = fmt.Sprintf(
				"🔄 Aguarde %02d:%02d até a próxima recarga (cooldown 5m).",
				minutos, segundos,
			)
			return m, nil
		}

		// Se estiver carregando ou não tiver playlists, ignoramos navegação
		if m.loading || len(m.playlists) == 0 {
			return m, nil
		}

		// Navegação normal: cursor move entre index 0 e len(playlists) (lembre: index 0 = “via URL”)
		switch msg.Type {
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			// índice máximo é len(playlists) (porque index 0 = “via URL”, playlists começam no 1)
			if m.cursor < len(m.playlists) {
				m.cursor++
			}
		case tea.KeyEnter:
			// Se cursor == 0, entramos no modo “digitar URL”
			if m.cursor == 0 {
				m.awaitingURL = true
				m.newURL = ""
				m.statusMessage = ""
				m.urlError = nil
				return m, nil
			}
			// Caso contrário, playlist selecionada (offset -1)
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

	// Se estivermos carregando playlists
	if m.loading {
		b.WriteString("Loading playlists…\n")
		b.WriteString("\n")
		// Mostra instrução de refresh
		b.WriteString(welcomePromptStyle.Render("Pressione Ctrl+R para recarregar (cooldown 5m). Ctrl+C para sair."))
		return docStyle.Render(b.String())
	}

	// Se estivermos aguardando o usuário digitar a URL
	if m.awaitingURL {
		b.WriteString("Digite a URL da playlist do YouTube e pressione Enter:\n")
		inputLine := fmt.Sprintf("> %s", m.newURL)
		b.WriteString(listItemStyle.Render(inputLine))
		b.WriteString("\n\n")
		if m.urlError != nil {
			b.WriteString(errorMessageStyle.Render(fmt.Sprintf("Erro: %v", m.urlError)))
			b.WriteString("\n\n")
		}
		b.WriteString(welcomePromptStyle.Render("Pressione Backspace para apagar, Enter para confirmar."))
		return docStyle.Render(b.String())
	}

	// Se ocorreu erro no carregamento normal
	if m.err != nil {
		b.WriteString(errorMessageStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
		b.WriteString(welcomePromptStyle.Render("Pressione Enter para voltar ao login."))
		b.WriteString("\n")
		b.WriteString(welcomePromptStyle.Render("Pressione Ctrl+R para recarregar. Ctrl+C para sair."))
		return docStyle.Render(b.String())
	}

	if m.cursor == 0 {
		b.WriteString(selectedListItemStyle.Render("Reordenar playlist via URL"))
	} else {
		b.WriteString(listItemStyle.Render("Reordenar playlist via URL"))
	}
	b.WriteString("\n")

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

	// Se tiver mensagem de status (por exemplo, cooldown), exiba abaixo
	if m.statusMessage != "" {
		b.WriteString("\n\n")
		b.WriteString(statusMessageStyle.Render(m.statusMessage))
	}

	return docStyle.Render(b.String())
}
