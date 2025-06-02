package token_manager

import (
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"os"
)

type tokenServiceImpl struct {
	TokenFilePath string
}

type TokenService interface {
	DeleteLocalToken() error
	LoadToken() (*oauth2.Token, error)
	SaveToken(token *oauth2.Token) error
}

func NewTokenService(tokenFilePath string) TokenService {
	if tokenFilePath == "" {
		tokenFilePath = "token.json"
	}

	return &tokenServiceImpl{
		TokenFilePath: tokenFilePath,
	}
}

func (t *tokenServiceImpl) DeleteLocalToken() error {
	err := os.Remove(t.TokenFilePath)
	if err != nil {
		return fmt.Errorf("não foi possível remover o arquivo de token: %w", err)
	}

	return nil
}

func (t *tokenServiceImpl) LoadToken() (*oauth2.Token, error) {
	file, err := os.Open(t.TokenFilePath)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir arquivo de token %s: %w", t.TokenFilePath, err)
	}

	defer file.Close()
	token := &oauth2.Token{}

	err = json.NewDecoder(file).Decode(token)
	if err != nil {
		return nil, fmt.Errorf("falha ao decodificar token do arquivo %s: %w", t.TokenFilePath, err)
	}

	if token.AccessToken == "" && token.RefreshToken == "" {
		return nil, fmt.Errorf("token inválido: não contém AccessToken ou RefreshToken")
	}

	return token, nil
}

func (t *tokenServiceImpl) SaveToken(token *oauth2.Token) error {
	file, err := os.OpenFile(t.TokenFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("não foi possível abrir/criar o arquivo de token %s: %w", t.TokenFilePath, err)
	}

	defer file.Close()
	return json.NewEncoder(file).Encode(token)
}
