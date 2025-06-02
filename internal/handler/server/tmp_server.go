package server

import (
	"TUI_playlist_reorder/infrastructure/logger"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type OAuthCallbackResult struct {
	Code  string
	Error error
}

type CallbackHandler interface {
	ListenAndServe(
		ctx context.Context,
		expectedState,
		addr,
		callbackPath string,
		resultChan chan<- OAuthCallbackResult,
	) *http.Server
}

type callbackHandlerImpl struct {
	logger logger.Logger
}

// NewCallbackHandler continua o mesmo
func NewCallbackHandler(logger logger.Logger) CallbackHandler {
	return &callbackHandlerImpl{
		logger: logger,
	}
}

func (h *callbackHandlerImpl) ListenAndServe(
	ctx context.Context,
	expectedState string,
	addr string,
	callbackPath string,
	resultChan chan<- OAuthCallbackResult,
) *http.Server {
	mux := http.NewServeMux()

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	handlerDone := make(chan struct{})

	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		defer close(handlerDone)

		state := r.URL.Query().Get("state")
		if state != expectedState {
			err := fmt.Errorf("state CSRF inválido. Recebido: '%s', Esperado: '%s'", state, expectedState)
			h.logger.Error("Erro de state CSRF: %v", err) // Exemplo de como seria com logger

			http.Error(w, "Invalid state. Please try the authentication process again.", http.StatusBadRequest)
			resultChan <- OAuthCallbackResult{Error: err}
			return
		}

		authErrParam := r.URL.Query().Get("error") // Renomeado para evitar conflito com a variável 'err'
		if authErrParam != "" {
			errDesc := r.URL.Query().Get("error_description")
			var errMsg error
			if errDesc != "" {
				errMsg = fmt.Errorf("erro de autorização do provedor OAuth: %s - %s", authErrParam, errDesc)
			} else {
				errMsg = fmt.Errorf("erro de autorização do provedor OAuth: %s", authErrParam)
			}

			h.logger.Error("Erro do provedor OAuth: %v", errMsg)

			http.Error(w, "An error occurred during authorization with the provider. You can close this tab.", http.StatusUnauthorized) // Resposta HTTP adicionada
			resultChan <- OAuthCallbackResult{Error: errMsg}
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			err := fmt.Errorf("código de autorização não encontrado na requisição de callback")
			h.logger.Warning("Código de autorização não encontrado.")

			http.Error(w, "Authorization code not found in the request.", http.StatusBadRequest)
			resultChan <- OAuthCallbackResult{Error: err}
			return
		}

		successMsg := "Autorização recebida com sucesso! Você pode fechar esta aba do navegador."
		fmt.Fprint(w, successMsg)
		h.logger.Info("Código de autorização recebido com sucesso.")

		resultChan <- OAuthCallbackResult{Code: code}
	})

	go func() {
		h.logger.Info("Iniciando servidor de callback em " + addr + " - " + callbackPath)

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			wrappedErr := fmt.Errorf("falha crítica ao iniciar servidor de callback HTTP: %w", err)
			h.logger.Error("%v", wrappedErr)

			select {
			case resultChan <- OAuthCallbackResult{Error: wrappedErr}:
			default:
			}
		}

		h.logger.Info("Servidor de callback HTTP: ListenAndServe retornou.")
	}()

	go func() {
		select {
		case <-handlerDone:
			h.logger.Info("Servidor de callback OAuth: Handler concluiu, iniciando shutdown.")
		case <-ctx.Done():
			h.logger.Info("Servidor de callback OAuth: Contexto cancelado/timeout " + ctx.Err().Error() + ", iniciando shutdown.")
		}

		h.logger.Info("Iniciando shutdown do servidor de callback...")
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelShutdown()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			h.logger.Error("Erro ao desligar servidor de callback HTTP: ", err)
		} else {
			h.logger.Info("Servidor de callback HTTP desligado com sucesso.")
		}
	}()

	return httpServer
}
