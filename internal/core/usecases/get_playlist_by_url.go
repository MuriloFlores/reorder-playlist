package usecases

import (
	"TUI_playlist_reorder/internal/core/domain"
	"context"
	"fmt"
)

func (uc *playlistUseCase) GetPlaylistByURL(ctx context.Context, url string) (domain.Playlist, error) {
	uc.log.Info("Init Get Playlist By URL")

	// Validate the URL
	if url == "" {
		return domain.Playlist{}, fmt.Errorf("playlist URL cannot be empty")
	}

	// Call the service to get the playlist by URL
	playlist, err := uc.service.GetPlaylistByURL(url, ctx)
	if err != nil {
		uc.log.Error("Failed to get playlist by URL", err)
		return domain.Playlist{}, err
	}

	uc.log.Info("Get Playlist By URL Completed")

	return playlist, nil
}
