package domain

import "sort"

type Playlist struct {
	ID        string
	ChannelID string
	Title     string
	Videos    []Video
}

func (p *Playlist) SortByName() {
	sort.Slice(p.Videos, func(i, j int) bool {
		return p.Videos[i].Title < p.Videos[j].Title
	})
}

func (p *Playlist) SortByDuration() {
	sort.Slice(p.Videos, func(i, j int) bool {
		return p.Videos[i].Duration < p.Videos[j].Duration
	})
}

func (p *Playlist) SortByPublish() {
	sort.Slice(p.Videos, func(i, j int) bool {
		return p.Videos[i].PublishedAt.Before(p.Videos[j].PublishedAt)
	})
}

func (p *Playlist) SortByLanguage() {
	sort.Slice(p.Videos, func(i, j int) bool {
		return p.Videos[i].Language < p.Videos[j].Language
	})
}
