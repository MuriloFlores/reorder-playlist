package usecases

import (
	"TUI_playlist_reorder/internal/core/domain"
	"TUI_playlist_reorder/internal/core/ports"
	"context"
)

type playlistUseCase struct {
	service ports.YoutubePort
	log     ports.LoggerPort
}

type PlaylistUseCase interface {
	GetMinePlaylists(ctx context.Context) ([]domain.Playlist, error)
	ReorderPlaylist(ctx context.Context, playlistID, criteria, title string) error
}

func NewPlaylistUseCase(service ports.YoutubePort, logger ports.LoggerPort) PlaylistUseCase {
	return &playlistUseCase{
		service: service,
		log:     logger,
	}
}
