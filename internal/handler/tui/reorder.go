package tui

import (
	"TUI_playlist_reorder/internal/core/domain"
	"TUI_playlist_reorder/internal/core/usecases"
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type ReorderModel struct {
	parent          *AppModel
	playlist        domain.Playlist
	playlistUseCase usecases.PlaylistUseCase

	// Opções de reordenação
	reorderOptions []string
	cursor         int

	// Campos para entrada de título
	awaitingTitle   bool   // true = usuário digitando o novo título
	pendingCriteria string // critério selecionado ("name", "duration", etc.)
	newTitle        string // texto digitado pelo usuário

	// Estado de “loading”
	awaitingSave bool // true enquanto o “timer de 10s” estiver rodando

	// Mensagens de status/erro
	statusMessage string
	err           error
}

func NewReorderModel(parent *AppModel, playlist domain.Playlist) *ReorderModel {
	return &ReorderModel{
		parent:          parent,
		playlist:        playlist,
		playlistUseCase: parent.playlistUseCase,
		reorderOptions: []string{
			"Ordenar por Nome (A-Z)",
			"Ordenar por Duração (Menor-Maior)",
			"Ordenar por Idioma (A-Z)",
			"Ordenar por Data de Publicação (Mais Antigo-Mais Novo)",
			"Voltar para Playlists",
		},
		cursor:          0,
		awaitingTitle:   false,
		pendingCriteria: "",
		newTitle:        "",
		awaitingSave:    false,
		statusMessage:   "",
		err:             nil,
	}
}

func (m *ReorderModel) Init() tea.Cmd {
	m.statusMessage = ""
	m.err = nil
	m.awaitingSave = false
	m.parent.logger.Info(fmt.Sprintf("ReorderModel: inicializado para playlist '%s'", m.playlist.Title))
	return nil
}

func (m *ReorderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// Se estivermos aguardando o “loading” de 10 segundos, ignoramos toda KeyMsg
	case tea.KeyMsg:
		if m.awaitingSave {
			// Enquanto o timer estiver rodando, não aceitamos input
			return m, nil
		}

		// Modo de digitar título
		if m.awaitingTitle {
			switch msg.Type {
			case tea.KeyEnter:
				// O usuário terminou de digitar o título
				m.awaitingTitle = false
				title := strings.TrimSpace(m.newTitle)
				if title == "" {
					m.err = fmt.Errorf("título não pode ser vazio")
					m.statusMessage = "Título não pode ficar vazio. Tente novamente."
					m.newTitle = ""
					return m, nil
				}

				// Atualiza localmente antes de aguardar o timer
				switch m.pendingCriteria {
				case "name":
					m.playlist.SortByName()
				case "duration":
					m.playlist.SortByDuration()
				case "language":
					m.playlist.SortByLanguage()
				case "publish":
					m.playlist.SortByPublish()
				}

				// Entra no modo “loading” de 10s
				m.awaitingSave = true
				m.err = nil
				m.statusMessage = fmt.Sprintf("Salvando playlist \"%s\" no YouTube. Aguarde...", title)

				// Após 10 segundos, dispara um TeaMsg que manda executar o salvamento real
				return m, tea.Tick(time.Second*10, func(t time.Time) tea.Msg {
					return savePlaylistMsg{criteria: m.pendingCriteria, title: title}
				})

			case tea.KeyBackspace:
				if len(m.newTitle) > 0 {
					m.newTitle = m.newTitle[:len(m.newTitle)-1]
				}
				return m, nil

			default:
				if msg.Type == tea.KeyRunes {
					m.newTitle += string(msg.Runes)
				}
				return m, nil
			}
		}

		// Modo normal (navegar opções)
		switch msg.Type {
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			if m.cursor < len(m.reorderOptions)-1 {
				m.cursor++
			}
		case tea.KeyEnter:
			selecionado := m.reorderOptions[m.cursor]
			switch selecionado {
			case "Ordenar por Nome (A-Z)":
				m.pendingCriteria = "name"
				m.awaitingTitle = true
				m.newTitle = ""
				m.statusMessage = ""
				m.err = nil

			case "Ordenar por Duração (Menor-Maior)":
				m.pendingCriteria = "duration"
				m.awaitingTitle = true
				m.newTitle = ""
				m.statusMessage = ""
				m.err = nil

			case "Ordenar por Idioma (A-Z)":
				m.pendingCriteria = "language"
				m.awaitingTitle = true
				m.newTitle = ""
				m.statusMessage = ""
				m.err = nil

			case "Ordenar por Data de Publicação (Mais Antigo-Mais Novo)":
				m.pendingCriteria = "publish"
				m.awaitingTitle = true
				m.newTitle = ""
				m.statusMessage = ""
				m.err = nil

			case "Voltar para Playlists":
				return m, m.parent.send(showPlaylistsMsg{})
			}
		case tea.KeyBackspace:
			return m, m.parent.send(showPlaylistsMsg{})
		}
	}

	// Mensagem customizada para, após o timer, efetivar o salvamento
	switch msg := msg.(type) {
	case savePlaylistMsg:
		m.awaitingSave = false

		// Executa o use case de fato
		err := m.playlistUseCase.ReorderPlaylist(context.Background(), m.playlist.ID, msg.criteria, msg.title)
		if err != nil {
			m.err = err
			m.statusMessage = fmt.Sprintf("Erro ao salvar no YouTube: %v", err)
		} else {
			m.err = nil
			m.statusMessage = "Playlist salva com sucesso no YouTube."
			m.playlist.Title = msg.title
		}
		return m, nil
	}

	return m, nil
}

func (m *ReorderModel) View() string {
	var b strings.Builder

	title := listHeaderStyle.Render(fmt.Sprintf("Reordenar Playlist: %s", m.playlist.Title))
	b.WriteString(title)
	b.WriteString("\n\n")

	// Se estivermos aguardando o timer de “loading”
	if m.awaitingSave {
		b.WriteString("⏳ ")
		b.WriteString(statusMessageStyle.Render(m.statusMessage))
		b.WriteString("\n\n")
		return docStyle.Render(b.String())
	}

	// Se estivermos pedindo para o usuário digitar título
	if m.awaitingTitle {
		b.WriteString("Digite o novo título para a playlist e pressione Enter:\n")
		inputLine := fmt.Sprintf("> %s", m.newTitle)
		b.WriteString(listItemStyle.Render(inputLine))
		b.WriteString("\n\n")
		if m.err != nil {
			b.WriteString(errorMessageStyle.Render(fmt.Sprintf("Erro: %v", m.err)))
			b.WriteString("\n\n")
		}
		return docStyle.Render(b.String())
	}

	// Caso normal: exibe vídeos e opções
	b.WriteString("Playlist atual:" + m.playlist.Title + "\n")

	b.WriteString("Opções de Reordenação:\n")
	for i, opt := range m.reorderOptions {
		if m.cursor == i {
			b.WriteString(selectedListItemStyle.Render(opt))
		} else {
			b.WriteString(listItemStyle.Render(opt))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if m.statusMessage != "" {
		b.WriteString(statusMessageStyle.Render(m.statusMessage))
		b.WriteString("\n")
	}
	if m.err != nil {
		b.WriteString(errorMessageStyle.Render(fmt.Sprintf("Erro: %v", m.err)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(welcomePromptStyle.Render("Use ↑/↓ para navegar, Enter para selecionar, Backspace para voltar. Ctrl+C para sair."))

	return docStyle.Render(b.String())
}

// Mensagem interna para disparar o salvamento após o timer
type savePlaylistMsg struct {
	criteria string
	title    string
}
