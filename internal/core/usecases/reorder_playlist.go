package usecases

import (
	"context"
	"fmt"
)

func (uc *playlistUseCase) ReorderPlaylist(ctx context.Context, playlistID, criteria, title string) error {
	uc.log.Info("Init Reorder Playlist")

	// Validate the playlist ID
	if playlistID == "" {
		return fmt.Errorf("playlist ID cannot be empty")
	}

	// Call the service to reorder the playlist
	playlist, err := uc.service.GetPlaylistByID(playlistID, ctx)
	if err != nil {
		uc.log.Error("Failed to reorder playlist", err)
		return fmt.Errorf("error while reordering playlist: %w", err)
	}

	// Reorder the playlist based on the criteria
	switch criteria {
	case "name":
		playlist.SortByName()
	case "duration":
		playlist.SortByDuration()
	case "publish":
		playlist.SortByPublish()
	case "language":
		playlist.SortByLanguage()
	}

	uc.log.Info("Reorder Playlist Completed")

	// Save the reordered playlist
	err = uc.service.SavePlaylist(title, playlist, ctx)
	if err != nil {
		uc.log.Error("Failed to save reordered playlist", err)
		return fmt.Errorf("error while saving reordered playlist: %w", err)
	}

	uc.log.Info("Reordered playlist saved successfully")

	return nil
}
