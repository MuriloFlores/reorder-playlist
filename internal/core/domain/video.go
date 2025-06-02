package domain

import "time"

type Video struct {
	ID          string
	Title       string
	Artist      string
	PublishedAt time.Time
	Duration    time.Duration
	Language    string
}
