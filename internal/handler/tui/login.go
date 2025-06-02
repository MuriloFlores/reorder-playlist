package tui

import (
	"context"
	"fmt"
	"golang.org/x/oauth2"
	"strings"
	"time"

	"TUI_playlist_reorder/infrastructure/auth"
	"TUI_playlist_reorder/infrastructure/logger"
	"TUI_playlist_reorder/internal/handler/server"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/browser"
)

type authURLGeneratedMsg struct{ url string }
type authSuccessMsg struct {
	token *oauth2.Token
	code  string
}
type authErrorMsg struct{ err error }

type loginState int

const (
	loginIdle loginState = iota
	loginAuthURLGenerated
	loginWaitingForCallback
	loginExchangingToken
	loginSuccess
	loginError
)

type LoginModel struct {
	parent           *AppModel
	state            loginState
	authURL          string
	errorMsg         string
	statusMsg        string
	csrfState        string
	httpServerCtx    context.Context
	httpServerCancel context.CancelFunc
}

func NewLoginModel(parent *AppModel) *LoginModel {
	return &LoginModel{
		parent:    parent,
		state:     loginIdle,
		statusMsg: "Pressione Enter para iniciar o login com Google...",
	}
}

func (m *LoginModel) Init() tea.Cmd {
	// Resetar estado a cada vez que a tela é apresentada
	m.state = loginIdle
	m.errorMsg = ""
	m.statusMsg = "Pressione Enter para iniciar o login com Google..."
	m.csrfState = fmt.Sprintf("st%d", time.Now().UnixNano())
	return nil
}

// → Gera URL de autenticação (executado em goroutine separada)
func generateAuthURLCmd(authService auth.AuthenticationService, state string) tea.Cmd {
	return func() tea.Msg {
		url := authService.GenerateAuthURL(state)
		return authURLGeneratedMsg{url: url}
	}
}

// → Espera callback do Google (em background)
func waitForCallbackCmd(
	ctx context.Context,
	callbackHandler server.CallbackHandler,
	expectedState string,
	addr string,
	callbackPath string,
	logger logger.Logger,
) tea.Cmd {
	return func() tea.Msg {
		resultChan := make(chan server.OAuthCallbackResult, 1)

		srvCtx, srvCancel := context.WithCancel(ctx)
		defer srvCancel()

		logger.Info(fmt.Sprintf("Iniciando servidor de callback em %s, estado esperado: %s", addr, expectedState))
		_ = callbackHandler.ListenAndServe(srvCtx, expectedState, addr, callbackPath, resultChan)

		logger.Info("Aguardando resultado do callback OAuth...")
		select {
		case res := <-resultChan:
			logger.Info(fmt.Sprintf("Resultado do callback: Code=%s, Err=%v", res.Code, res.Error))
			if res.Error != nil {
				return authErrorMsg{err: fmt.Errorf("callback error: %w", res.Error)}
			}
			if res.Code != "" {
				return authSuccessMsg{code: res.Code}
			}
			return authErrorMsg{err: fmt.Errorf("nenhum código ou erro no callback")}
		case <-ctx.Done():
			logger.Info("Callback cancelado pelo contexto principal.")
			return authErrorMsg{err: fmt.Errorf("login cancelado: %w", ctx.Err())}
		}
	}
}

// → Troca o código recebido por um token
func exchangeCodeCmd(authService auth.AuthenticationService, code string, appCtx context.Context) tea.Cmd {
	return func() tea.Msg {
		token, err := authService.ExchangeCodeForToken(appCtx, code)
		if err != nil {
			return authErrorMsg{err: fmt.Errorf("falha na troca de token: %w", err)}
		}
		return authSuccessMsg{token: token, code: ""}
	}
}

func (m *LoginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Se estamos no estado inicial ou em erro, Enter gera URL de login
		if (m.state == loginIdle || m.state == loginError) && msg.Type == tea.KeyEnter {
			m.state = loginAuthURLGenerated
			m.statusMsg = "Gerando URL de autenticação..."
			m.errorMsg = ""
			return m, generateAuthURLCmd(m.parent.authService, m.csrfState)
		}

	case authURLGeneratedMsg:
		m.authURL = msg.url
		m.statusMsg = "Abra este link no seu navegador para autenticar:\n"
		// Tenta abrir o navegador automaticamente
		go func() {
			if err := browser.OpenURL(m.authURL); err != nil {
				m.parent.logger.Error("Não foi possível abrir o navegador", err)
			}
		}()

		m.state = loginWaitingForCallback

		// Cria subcontexto para o servidor de callback
		serverCtx, serverCancel := context.WithCancel(m.parent.appContext)
		m.httpServerCtx = serverCtx
		m.httpServerCancel = serverCancel

		return m, waitForCallbackCmd(
			m.httpServerCtx,
			m.parent.callbackHandler,
			m.csrfState,
			":8080",
			"/",
			m.parent.logger,
		)

	case authSuccessMsg:
		// Se houve httpServerCancel, encerra o servidor de callback
		if m.httpServerCancel != nil {
			m.httpServerCancel()
			m.httpServerCancel = nil
			m.parent.logger.Info("Servidor de callback finalizado.")
		}

		// Se msg.code != "" → fase 1 (recebemos o código, trocar por token)
		if msg.code != "" {
			m.state = loginExchangingToken
			m.statusMsg = "Código recebido! Trocando por token..."
			return m, exchangeCodeCmd(m.parent.authService, msg.code, m.parent.appContext)
		}

		// msg.code == "" e token != nil → fase 2 (login concluído)
		if msg.token != nil {
			m.state = loginSuccess
			m.statusMsg = "Login bem-sucedido! Carregando playlists..."
			m.errorMsg = ""
			// Após breve atraso, navegar para playlists
			return m, tea.Sequence(
				tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg { return nil }),
				m.parent.send(showPlaylistsMsg{}),
			)
		}

	case authErrorMsg:
		// Em caso de erro, encerra o servidor de callback se estiver rodando
		if m.httpServerCancel != nil {
			m.httpServerCancel()
			m.httpServerCancel = nil
			m.parent.logger.Info("Servidor de callback encerrado devido a erro.")
		}
		m.state = loginError
		m.errorMsg = fmt.Sprintf("Falha no login: %v", msg.err)
		m.statusMsg = "Pressione Enter para tentar novamente."
		m.parent.logger.Error("authErrorMsg recebido", msg.err)
	}

	return m, nil
}

func (m *LoginModel) View() string {
	var b strings.Builder

	title := welcomeTitleStyle.Render("Google Authentication")
	b.WriteString(title)
	b.WriteString("\n\n")

	if m.errorMsg != "" {
		b.WriteString(errorMessageStyle.Render(m.errorMsg))
		b.WriteString("\n\n")
	}

	b.WriteString(m.statusMsg)
	b.WriteString("\n")

	if m.state == loginWaitingForCallback && m.authURL != "" {
		b.WriteString(urlStyle.Render(m.authURL))
		b.WriteString("\n\n")
		b.WriteString(welcomePromptStyle.Render("Aguardando a autenticação no navegador..."))
	}

	b.WriteString("\n\n")
	b.WriteString(welcomePromptStyle.Render("(Ctrl+C ou Esc para sair)"))
	return docStyle.Render(b.String())
}
