package ports

import (
	"TUI_playlist_reorder/internal/core/domain"
	"context"
)

type YoutubePort interface {
	GetAllPlaylistsFromUser(ctx context.Context) ([]domain.Playlist, error)
	GetPlaylistWithoutVideos(ctx context.Context) ([]domain.Playlist, error)
	GetPlaylistByID(playlistID string, ctx context.Context) (domain.Playlist, error)
	GetPlaylistByURL(playlistURL string, ctx context.Context) (domain.Playlist, error)
	DeletePlaylist(playlistID string, ctx context.Context) error
	SavePlaylist(title string, playlist domain.Playlist, ctx context.Context) error
}
