package provider

import (
	"TUI_playlist_reorder/infrastructure/token_manager"
	"TUI_playlist_reorder/internal/core/domain"
	"TUI_playlist_reorder/internal/core/ports"
	"context"
	"fmt"
	"github.com/sosodev/duration"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
	"net/url"
	"sync"
	"time"
)

type youtubeProvider struct {
	tokenService token_manager.TokenService
	log          ports.LoggerPort
	service      *youtube.Service
	mu           sync.Mutex
}

func NewYoutubeProvider(tokenService token_manager.TokenService, logger ports.LoggerPort) ports.YoutubePort {
	return &youtubeProvider{
		tokenService: tokenService,
		log:          logger,
		service:      nil,
		mu:           sync.Mutex{},
	}
}

func (s *youtubeProvider) getYoutubeService(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, err := s.tokenService.LoadToken()
	if err != nil {
		s.log.Error("error while load token: %w", err)
		return fmt.Errorf("error while load token: %w", err)
	}

	s.log.Info("Load token completed")

	service, err := youtube.NewService(ctx, option.WithTokenSource(oauth2.StaticTokenSource(token)))
	if err != nil {
		s.log.Error("error while create youtube service: %w", err)
		return fmt.Errorf("error while create youtube service: %w", err)
	}

	s.service = service

	s.log.Info("Create youtube service completed")

	return nil
}

func (s *youtubeProvider) GetAllPlaylistsFromUser(ctx context.Context) ([]domain.Playlist, error) {
	if s.service == nil {
		err := s.getYoutubeService(ctx)
		if err != nil {
			s.log.Error("error while get youtube service: %w", err)
			return nil, fmt.Errorf("error while create youtube provider: %w", err)
		}
	}

	//preparando chamada para a api do YouTube
	call := s.service.Playlists.List([]string{"id", "snippet", "contentDetails"}).Mine(true).MaxResults(50)

	//realizando a chamada para a api
	response, err := call.Do()
	if err != nil {
		s.log.Error("error while call youtube service: %w", err)
		return nil, fmt.Errorf("error in call youtube api: %w", err)
	}

	//verifica se a resposta veio vazia
	if len(response.Items) == 0 {
		s.log.Warning("No youtube playlists found")
		return []domain.Playlist{}, nil
	}

	//convertendo a resposta da api para um vetor de playlist(domain)
	playlistDomain := make([]domain.Playlist, len(response.Items))
	for i, item := range response.Items {
		videos, err := s.getPlaylistVideos(item.Id, ctx)
		if err != nil {
			s.log.Error("error while get videos: %w", err)
			return nil, fmt.Errorf("error in getPlaylistvideos while get videos: %w", err)
		}

		playlistDomain[i] = domain.Playlist{
			ID:        item.Id,
			ChannelID: item.Snippet.ChannelId,
			Title:     item.Snippet.Title,
			Videos:    videos,
		}
	}

	defer s.log.Info("Get all playlists completed")

	return playlistDomain, nil
}

func (s *youtubeProvider) GetPlaylistWithoutVideos(ctx context.Context) ([]domain.Playlist, error) {
	if s.service == nil {
		err := s.getYoutubeService(ctx)
		if err != nil {
			s.log.Error("error while get youtube service: %w", err)
			return nil, fmt.Errorf("error while create youtube provider: %w", err)
		}
	}

	//preparando chamada para a api do YouTube
	call := s.service.Playlists.List([]string{"id", "snippet", "contentDetails"}).Mine(true).MaxResults(50)

	//realizando a chamada para a api
	response, err := call.Do()
	if err != nil {
		s.log.Error("error while call youtube service: %w", err)
		return nil, fmt.Errorf("error in call youtube api: %w", err)
	}

	//verifica se a resposta veio vazia
	if len(response.Items) == 0 {
		s.log.Warning("No youtube playlists found")
		return []domain.Playlist{}, nil
	}

	//convertendo a resposta da api para um vetor de playlist(domain)
	playlistDomain := make([]domain.Playlist, len(response.Items))
	for i, item := range response.Items {
		// Não busca os vídeos, apenas retorna a playlist sem vídeos
		playlistDomain[i] = domain.Playlist{
			ID:        item.Id,
			ChannelID: item.Snippet.ChannelId,
			Title:     item.Snippet.Title,
			Videos:    nil,
		}
	}

	defer s.log.Info("Get all playlists completed")

	return playlistDomain, nil
}

func (s *youtubeProvider) GetPlaylistByURL(playlistURL string, ctx context.Context) (domain.Playlist, error) {
	if s.service == nil {
		err := s.getYoutubeService(ctx)
		if err != nil {
			s.log.Error("error while get youtube service: %w", err)
			return domain.Playlist{}, fmt.Errorf("error while create youtube provider: %w", err)
		}
	}

	parsedURL, err := url.Parse(playlistURL)
	if err != nil {
		return domain.Playlist{}, fmt.Errorf("error in parsing playlist url: %w", err)
	}

	//buscando o playlist id baseado no parâmetro list da url
	playlistID := parsedURL.Query().Get("list")
	if playlistID == "" {
		return domain.Playlist{}, fmt.Errorf("playlist id not exists")
	}

	//pegando os dados dos videos
	videos, err := s.getPlaylistVideos(playlistID, ctx)
	if err != nil {
		return domain.Playlist{}, fmt.Errorf("error in getPlaylistvideos while get videos: %w", err)
	}

	//preparando a chamada para a api
	call := s.service.Playlists.List([]string{"id", "snippet"}).Id(playlistID)
	response, err := call.Do()
	if err != nil {
		return domain.Playlist{}, fmt.Errorf("error in call youtube api: %w", err)
	}

	//preparando o domain para retornar
	playlistDomain := domain.Playlist{
		ID:        response.Items[0].Id,
		ChannelID: response.Items[0].Snippet.ChannelId,
		Title:     response.Items[0].Snippet.Title,
		Videos:    videos,
	}

	return playlistDomain, nil
}

func (s *youtubeProvider) GetPlaylistByID(playlistID string, ctx context.Context) (domain.Playlist, error) {
	if s.service == nil {
		err := s.getYoutubeService(ctx)
		if err != nil {
			s.log.Error("error while get youtube service: %w", err)
			return domain.Playlist{}, fmt.Errorf("error while create youtube provider: %w", err)
		}
	}

	//chama a api do youtube para pegar os dados da playlist
	call := s.service.Playlists.List([]string{"id", "snippet"}).Id(playlistID)
	response, err := call.Do()
	if err != nil {
		return domain.Playlist{}, fmt.Errorf("error in call youtube api: %w", err)
	}

	//verifica se a resposta veio vazia
	if len(response.Items) == 0 {
		return domain.Playlist{}, fmt.Errorf("playlist not found")
	}

	//pega os videos da playlist
	videos, err := s.getPlaylistVideos(playlistID, ctx)
	if err != nil {
		return domain.Playlist{}, fmt.Errorf("error in getPlaylistvideos while get videos: %w", err)
	}

	item := response.Items[0]

	//preparando o domain para retornar
	playlistDomain := domain.Playlist{
		ID:        item.Id,
		ChannelID: item.Snippet.ChannelId,
		Title:     item.Snippet.Title,
		Videos:    videos,
	}

	return playlistDomain, nil
}

func (s *youtubeProvider) DeletePlaylist(playlistID string, ctx context.Context) error {
	if s.service == nil {
		err := s.getYoutubeService(ctx)
		if err != nil {
			return fmt.Errorf("error while create youtube service: %w", err)
		}
	}

	err := s.service.Playlists.Delete(playlistID).Do()
	if err != nil {
		return fmt.Errorf("error in call youtube api: %w", err)
	}

	return nil
}

func (s *youtubeProvider) SavePlaylist(title string, playlist domain.Playlist, ctx context.Context) error {
	if s.service == nil {
		if err := s.getYoutubeService(ctx); err != nil {
			return fmt.Errorf("error while create youtube service: %w", err)
		}
	}

	insertCall := s.service.Playlists.Insert(
		[]string{"snippet", "status"},
		&youtube.Playlist{
			Snippet: &youtube.PlaylistSnippet{
				Title:       title,
				Description: "",
			},
			Status: &youtube.PlaylistStatus{
				PrivacyStatus: "public",
			},
		},
	).Context(ctx)

	newPlaylist, err := insertCall.Do()
	if err != nil {
		return fmt.Errorf("error while create playlist: %w", err)
	}

	newPlaylistID := newPlaylist.Id
	s.log.Info(fmt.Sprintf("Playlist criada no YouTube com ID: %s", newPlaylistID))

	for _, video := range playlist.Videos {
		if err := s.addVideoToPlaylist(newPlaylistID, video.ID, ctx); err != nil {
			return fmt.Errorf("error while insert video in playlist: %w", err)
		}
	}

	return nil
}

func (s *youtubeProvider) addVideoToPlaylist(playlistID, videoID string, ctx context.Context) error {
	if s.service == nil {
		err := s.getYoutubeService(ctx)
		if err != nil {
			return fmt.Errorf("error while create youtube service: %w", err)
		}
	}

	upload := &youtube.PlaylistItem{
		Snippet: &youtube.PlaylistItemSnippet{
			PlaylistId: playlistID,
			ResourceId: &youtube.ResourceId{
				Kind:    "youtube#video",
				VideoId: videoID,
			},
		},
	}

	call := s.service.PlaylistItems.Insert([]string{"id", "snippet", "contentDetails"}, upload)

	_, err := call.Do()
	if err != nil {
		return fmt.Errorf("error in call youtube api: %w", err)
	}

	return nil
}

func (s *youtubeProvider) getPlaylistVideos(playlistID string, ctx context.Context) ([]domain.Video, error) {
	pageToken := ""
	var videos []domain.Video

	for {
		returnedVideos, nextPageToken, err := s.enrich(playlistID, pageToken, ctx)
		if err != nil {
			return nil, fmt.Errorf("error while getting youtube videos: %w", err)
		}

		videos = append(videos, returnedVideos...)

		if nextPageToken == "" {
			break
		}

		pageToken = nextPageToken
	}

	if len(videos) == 0 {
		return nil, fmt.Errorf("no youtube video found for playlist %s", playlistID)
	}

	return videos, nil
}

func (s *youtubeProvider) enrich(playlistID, pageToken string, ctx context.Context) ([]domain.Video, string, error) {
	if s.service == nil {
		err := s.getYoutubeService(ctx)
		if err != nil {
			return []domain.Video{}, "", fmt.Errorf("error while create youtube service: %w", err)
		}
	}

	//preparando a chamada a api, onde devera retornar os itens da playlist baseado no "id" da playlist a pesquisa usa o pageToken para paginação
	call := s.service.PlaylistItems.List([]string{"id", "contentDetails"}).PlaylistId(playlistID).PageToken(pageToken)

	//realizando a chamada
	response, err := call.Do()
	if err != nil {
		return []domain.Video{}, "", fmt.Errorf("error in call youtube api: %w", err)
	}

	// Cria um slice vazio com capacidade baseada na quantidade de itens retornados
	videos := make([]domain.Video, 0, len(response.Items))

	//iterando os itens da resposta e criando os videos(domain)
	for _, item := range response.Items {
		if item.ContentDetails == nil || item.ContentDetails.VideoId == "" {
			continue
		}

		video, err := s.getVideoDetails(item.ContentDetails.VideoId, ctx)
		if err != nil {
			continue
		}

		videos = append(videos, video)
	}

	return videos, response.NextPageToken, nil
}

func (s *youtubeProvider) getVideoDetails(videoID string, ctx context.Context) (domain.Video, error) {
	if s.service == nil {
		err := s.getYoutubeService(ctx)
		if err != nil {
			return domain.Video{}, fmt.Errorf("error while create youtube service: %w", err)
		}
	}

	call := s.service.Videos.List([]string{"snippet", "contentDetails"}).Id(videoID)
	response, err := call.Do()

	if err != nil {
		return domain.Video{}, fmt.Errorf("error while getting tack info: %w", err)
	}

	if len(response.Items) == 0 {
		return domain.Video{}, fmt.Errorf("video not found")
	}

	item := response.Items[0]

	parseDuration, err := duration.Parse(item.ContentDetails.Duration)
	if err != nil {
		return domain.Video{}, fmt.Errorf("error while parsing video duration: %w", err)
	}

	parsePublish, err := time.Parse(time.RFC3339, item.Snippet.PublishedAt)
	if err != nil {
		return domain.Video{}, fmt.Errorf("error while parsing video published: %w", err)
	}

	video := domain.Video{
		ID:          item.Id,
		Title:       item.Snippet.Title,
		Artist:      item.Snippet.ChannelTitle,
		PublishedAt: parsePublish,
		Duration:    parseDuration.ToTimeDuration(),
		Language:    item.Snippet.DefaultAudioLanguage,
	}

	return video, nil
}
