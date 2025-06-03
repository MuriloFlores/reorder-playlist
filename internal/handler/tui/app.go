package tui

import (
	"fmt"

	"TUI_playlist_reorder/infrastructure/auth"
	"TUI_playlist_reorder/infrastructure/logger"
	"TUI_playlist_reorder/infrastructure/token_manager"
	"TUI_playlist_reorder/internal/core/domain"
	"TUI_playlist_reorder/internal/core/usecases"
	"TUI_playlist_reorder/internal/handler/server"
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

type currentView int

const (
	viewWelcome currentView = iota
	viewLogin
	viewPlaylists
	viewReorder
	viewURL
)

type AppModel struct {
	// Dependências injetadas
	authService     auth.AuthenticationService
	callbackHandler server.CallbackHandler
	playlistUseCase usecases.PlaylistUseCase
	tokenService    token_manager.TokenService
	logger          logger.Logger

	welcomeModel   *WelcomeModel
	loginModel     *LoginModel
	playlistsModel *PlaylistsModel
	reorderModel   *ReorderModel
	urlModel       *URLModel

	currentView currentView
	err         error

	appContext context.Context
	cancelApp  context.CancelFunc

	width  int
	height int
}

func NewAppModel(
	authSvc auth.AuthenticationService,
	cbHandler server.CallbackHandler,
	playlistUC usecases.PlaylistUseCase,
	tokenSvc token_manager.TokenService,
	log logger.Logger,
) *AppModel {
	// Cria contexto principal que será cancelado no Quit
	appCtx, cancel := context.WithCancel(context.Background())

	m := &AppModel{
		authService:     authSvc,
		callbackHandler: cbHandler,
		playlistUseCase: playlistUC,
		tokenService:    tokenSvc,
		logger:          log,

		appContext: appCtx,
		cancelApp:  cancel,
	}

	// Cria instâncias de cada modelo, injetando ponteiro para AppModel
	m.welcomeModel = NewWelcomeModel(m)
	m.loginModel = NewLoginModel(m)
	m.playlistsModel = NewPlaylistsModel(m)
	m.urlModel = NewURLModel(m)
	m.reorderModel = NewReorderModel(m, domain.Playlist{})

	m.currentView = viewWelcome
	return m
}

func (m *AppModel) Init() tea.Cmd {
	// No início, queremos verificar se já existe token salvo
	return func() tea.Msg {
		_, err := m.tokenService.LoadToken()
		if err == nil {
			// Se houver token válido, vai direto para playlists
			m.logger.Info("Token existente encontrado, navegando para Playlists")
			return showPlaylistsMsg{}
		}
		// Senão, mostra boas-vindas
		m.logger.Info("Nenhum token válido, mostrando Welcome")
		return showWelcomeMsg{}
	}
}

// Mensagens de navegação que os sub-modelos usam
type showWelcomeMsg struct{}
type showLoginMsg struct{}
type showPlaylistsMsg struct{}
type showReorderMsg struct{ playlist domain.Playlist }
type showURLMsg struct{}

func (m *AppModel) send(msg tea.Msg) tea.Cmd {
	return func() tea.Msg { return msg }
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Tratamos keys globais (Ctrl+C, Esc)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.logger.Info("Ctrl+C ou Esc pressionado, encerrando app.")
			m.cancelApp()
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Propaga para sub-modelos (para que eles reajustem se quiserem)
		// WelcomeModel
		if m.welcomeModel != nil {
			updatedWelcome, _ := m.welcomeModel.Update(msg)
			if casted, ok := updatedWelcome.(*WelcomeModel); ok {
				m.welcomeModel = casted
			}
		}
		// LoginModel
		if m.loginModel != nil {
			updatedLogin, _ := m.loginModel.Update(msg)
			if casted, ok := updatedLogin.(*LoginModel); ok {
				m.loginModel = casted
			}
		}
		// PlaylistsModel
		if m.playlistsModel != nil {
			updatedPlaylists, _ := m.playlistsModel.Update(msg)
			if casted, ok := updatedPlaylists.(*PlaylistsModel); ok {
				m.playlistsModel = casted
			}
		}
		// ReorderModel
		if m.reorderModel != nil {
			updatedReorder, _ := m.reorderModel.Update(msg)
			if casted, ok := updatedReorder.(*ReorderModel); ok {
				m.reorderModel = casted
			}
		}
	}

	// Processa mensagens de navegação (ou de erro geral)
	switch msg := msg.(type) {
	case showWelcomeMsg:
		m.currentView = viewWelcome
		m.err = nil
		cmd = m.welcomeModel.Init()

	case showLoginMsg:
		m.currentView = viewLogin
		m.err = nil
		cmd = m.loginModel.Init()

	case showPlaylistsMsg:
		m.currentView = viewPlaylists
		m.err = nil
		// Toda vez que chegar aqui, recria o PlaylistsModel para refazer o fetch
		pm := NewPlaylistsModel(m)
		m.playlistsModel = pm
		cmd = pm.Init()

	case showReorderMsg:
		m.currentView = viewReorder
		m.err = nil
		rm := NewReorderModel(m, msg.playlist)
		m.reorderModel = rm
		cmd = rm.Init()

	case showURLMsg:
		m.currentView = viewURL
		m.err = nil
		um := NewURLModel(m)
		m.urlModel = um
		cmd = um.Init()
	}

	cmds = append(cmds, cmd)

	// Agora delegamos o Update ao submodel correto, de acordo com a tela atual
	var currentViewCmd tea.Cmd
	switch m.currentView {
	case viewWelcome:
		if m.welcomeModel != nil {
			updated, cmd := m.welcomeModel.Update(msg)
			if casted, ok := updated.(*WelcomeModel); ok {
				m.welcomeModel = casted
			}
			currentViewCmd = cmd
		}

	case viewLogin:
		if m.loginModel != nil {
			updated, cmd := m.loginModel.Update(msg)
			if casted, ok := updated.(*LoginModel); ok {
				m.loginModel = casted
			}
			currentViewCmd = cmd
		}

	case viewPlaylists:
		if m.playlistsModel != nil {
			updated, cmd := m.playlistsModel.Update(msg)
			if casted, ok := updated.(*PlaylistsModel); ok {
				m.playlistsModel = casted
			}
			currentViewCmd = cmd
		}

	case viewReorder:
		if m.reorderModel != nil {
			updated, cmd := m.reorderModel.Update(msg)
			if casted, ok := updated.(*ReorderModel); ok {
				m.reorderModel = casted
			}
			currentViewCmd = cmd
		}

	case viewURL:
		if m.urlModel != nil {
			updated, cmd := m.urlModel.Update(msg)
			if casted, ok := updated.(*URLModel); ok {
				m.urlModel = casted
			}
			currentViewCmd = cmd
		}

	}

	cmds = append(cmds, currentViewCmd)
	return m, tea.Batch(cmds...)
}

func (m *AppModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Ocorreu um erro: %v\n\n(Ctrl+C para sair)", m.err)
	}

	switch m.currentView {
	case viewWelcome:
		return m.welcomeModel.View()
	case viewLogin:
		return m.loginModel.View()
	case viewPlaylists:
		return m.playlistsModel.View()
	case viewReorder:
		return m.reorderModel.View()
	case viewURL:
		return m.urlModel.View()
	default:
		return "Visão desconhecida…"
	}
}
