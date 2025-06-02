// main.go
package main

import (
	"TUI_playlist_reorder/infrastructure/auth"
	"TUI_playlist_reorder/infrastructure/logger"
	"TUI_playlist_reorder/infrastructure/provider"
	"TUI_playlist_reorder/internal/handler/server"
	"TUI_playlist_reorder/internal/handler/tui"

	"TUI_playlist_reorder/infrastructure/token_manager"
	"TUI_playlist_reorder/internal/core/usecases"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"google.golang.org/api/youtube/v3" // For scopes
)

const (
	clientSecretFilePath = "./infrastructure/auth/client_secret.json"
	tokenFilePath        = "./infrastructure/token_manager/token.json"
	callbackURL          = "http://localhost:8080"
)

func main() {
	// Initialize Logger
	appLogger, err := logger.NewFileLogger("logs", "reorder_playlist_tui")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer appLogger.Close()
	appLogger.Info("Application starting...")

	// Initialize Services
	tokenService := token_manager.NewTokenService(tokenFilePath)

	authService, err := auth.NewAuthenticationService(
		[]string{youtube.YoutubeScope}, // Added youtube.YoutubeScope
		clientSecretFilePath,
		callbackURL,
		tokenService,
	)
	if err != nil {
		appLogger.Error("Failed to initialize auth service", err)
		fmt.Fprintf(os.Stderr, "Failed to initialize auth service: %v\n", err)
		os.Exit(1)
	}

	callbackHandler := server.NewCallbackHandler(appLogger)
	youtubeProvider := provider.NewYoutubeProvider(tokenService, appLogger)
	playlistUseCase := usecases.NewPlaylistUseCase(youtubeProvider, appLogger)

	// Create the initial TUI model
	initialModel := tui.NewAppModel(authService, callbackHandler, playlistUseCase, tokenService, appLogger)

	// Start Bubble Tea program
	p := tea.NewProgram(initialModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		appLogger.Error("Error running TUI program", err)
		fmt.Fprintf(os.Stderr, "Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}
	appLogger.Info("Application finished.")
}
