package usecases

import (
	"TUI_playlist_reorder/internal/core/domain"
	"context"
	"fmt"
)

func (uc *playlistUseCase) GetMinePlaylists(ctx context.Context) ([]domain.Playlist, error) {
	uc.log.Info("Init Get my playlists")

	playlists, err := uc.service.GetPlaylistWithoutVideos(ctx)
	if err != nil {
		uc.log.Error("Failed to get playlists from user", err)
		return nil, fmt.Errorf("error while getting playlists from user: %w", err)
	}

	defer uc.log.Info("Get my playlists done")

	return playlists, nil
}
