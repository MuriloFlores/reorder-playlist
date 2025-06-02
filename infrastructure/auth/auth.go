package auth

import (
	"TUI_playlist_reorder/infrastructure/token_manager"
	"context"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"net/http"
	"net/url"
	"os"
)

type authenticationServiceImpl struct {
	clienteSecretFilePath string
	defaultRedirectURL    string
	oauthConfig           *oauth2.Config
	tokenService          token_manager.TokenService
}

type AuthenticationService interface {
	GetAuthenticatedClient(ctx context.Context) (*http.Client, *oauth2.Token, error)
	GenerateAuthURL(state string) string
	RevokeToken(tokenToRevoke string) error
	ExchangeCodeForToken(ctx context.Context, code string) (*oauth2.Token, error)
	//RevokeToken(ctx context.Context, token *oauth2.Token) error
}

func NewAuthenticationService(scopes []string, clienteSecretFilePath, redirectURL string, tokenServer token_manager.TokenService) (AuthenticationService, error) {
	config, err := loadConfig(scopes, clienteSecretFilePath)
	if err != nil {
		return nil, fmt.Errorf("não foi possível carregar a configuração do cliente: %w", err)
	}

	config.RedirectURL = redirectURL

	return &authenticationServiceImpl{
		clienteSecretFilePath: clienteSecretFilePath,
		defaultRedirectURL:    redirectURL,
		tokenService:          tokenServer,
		oauthConfig:           config,
	}, nil
}

func loadConfig(scopes []string, clienteSecretFilePath string) (*oauth2.Config, error) {
	b, err := os.ReadFile(clienteSecretFilePath)
	if err != nil {
		return nil, fmt.Errorf("não foi possível ler o arquivo de segredo do cliente (%s): %w", clienteSecretFilePath, err)
	}

	config, err := google.ConfigFromJSON(b, scopes...)
	if err != nil {
		return nil, fmt.Errorf("não foi possível analisar a configuração do cliente a partir do arquivo JSON: %w", err)
	}

	return config, nil
}

func (a *authenticationServiceImpl) GetAuthenticatedClient(ctx context.Context) (*http.Client, *oauth2.Token, error) {
	token, err := a.tokenService.LoadToken()
	if err != nil {
		return nil, nil, fmt.Errorf("não foi possível carregar o token: %w", err)
	}

	tokenSource := a.oauthConfig.TokenSource(ctx, token)
	refreshedToken, err := tokenSource.Token()
	if err != nil {
		_ = a.tokenService.DeleteLocalToken()
		return nil, nil, fmt.Errorf("não foi possível atualizar o token: %w", err)
	}

	if refreshedToken.AccessToken != token.AccessToken || (refreshedToken.RefreshToken != "" && refreshedToken.RefreshToken != token.RefreshToken) {
		if errSave := a.tokenService.SaveToken(refreshedToken); errSave != nil {
			return nil, nil, fmt.Errorf("não foi possível salvar o token atualizado: %w", errSave)
		}
	}

	return oauth2.NewClient(ctx, tokenSource), refreshedToken, nil
}

func (a *authenticationServiceImpl) GenerateAuthURL(state string) string {
	return a.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (a *authenticationServiceImpl) ExchangeCodeForToken(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := a.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("não foi possível trocar o código de autorização por um token: %w", err)
	}

	if err = a.tokenService.SaveToken(token); err != nil {
		return nil, fmt.Errorf("não foi possível salvar o token: %w", err)
	}

	return token, nil
}

func (a *authenticationServiceImpl) RevokeToken(tokenToRevoke string) error {
	if tokenToRevoke == "" {
		return nil
	}

	revokeURL := "https://oauth2.googleapis.com/revoke"

	data := url.Values{}
	data.Set("token", tokenToRevoke)

	resp, err := http.PostForm(revokeURL, data)
	if err != nil {
		return fmt.Errorf("falha ao enviar requisição de revogação de token: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("falha ao revogar o token, status: %s", resp.Status)
	}

	return nil
}
