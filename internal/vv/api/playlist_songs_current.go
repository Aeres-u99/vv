package api

import (
	"context"
	"net/http"
)

type MPDPlaylistSongsCurrent interface {
	CurrentSong(context.Context) (map[string][]string, error)
}

type PlaylistSongsCurrentHandler struct {
	mpd      MPDPlaylistSongsCurrent
	cache    *cache
	songHook func(map[string][]string) map[string][]string
}

func NewPlaylistSongsCurrentHandler(mpd MPDPlaylistSongsCurrent, songHook func(map[string][]string) map[string][]string) (*PlaylistSongsCurrentHandler, error) {
	c, err := newCache(map[string][]string{})
	if err != nil {
		return nil, err
	}
	return &PlaylistSongsCurrentHandler{
		mpd:      mpd,
		cache:    c,
		songHook: songHook,
	}, nil
}

func (a *PlaylistSongsCurrentHandler) Update(ctx context.Context) error {
	l, err := a.mpd.CurrentSong(ctx)
	if err != nil {
		return err
	}
	_, err = a.cache.SetIfModified(a.songHook(l))
	return err
}

func (a *PlaylistSongsCurrentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.cache.ServeHTTP(w, r)
}

func (a *PlaylistSongsCurrentHandler) Changed() <-chan struct{} {
	return a.cache.Changed()
}

func (a *PlaylistSongsCurrentHandler) Close() {
	a.cache.Close()
}
