package api

import (
	"context"
	"net/http"
	"sync"
)

type MPDPlaylistSongsAPI interface {
	PlaylistInfo(context.Context) ([]map[string][]string, error)
}

func NewPlaylistSongs(mpd MPDPlaylistSongsAPI, songsHook func([]map[string][]string) []map[string][]string) (*PlaylistSongs, error) {
	cache, err := newCache([]map[string][]string{})
	if err != nil {
		return nil, err
	}
	return &PlaylistSongs{
		mpd:       mpd,
		cache:     cache,
		changed:   make(chan struct{}, cap(cache.Changed())),
		songsHook: songsHook,
	}, nil

}

type PlaylistSongs struct {
	mpd        MPDPlaylistSongsAPI
	cache      *cache
	changed    chan struct{}
	songsHook  func([]map[string][]string) []map[string][]string
	eventHooks []func([]map[string][]string)
	data       []map[string][]string
	mu         sync.RWMutex
}

func (a *PlaylistSongs) Update(ctx context.Context) error {
	l, err := a.mpd.PlaylistInfo(ctx)
	if err != nil {
		return err
	}
	v := a.songsHook(l)
	changed, err := a.cache.SetIfModified(v)
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.data = v
	a.mu.Unlock()
	if changed {
		select {
		case a.changed <- struct{}{}:
		default:
		}
	}
	return nil
}

func (a *PlaylistSongs) Cache() []map[string][]string {
	a.mu.RLock()
	a.mu.RUnlock()
	return a.data
}

// ServeHTTP responses neighbors list as json format.
func (a *PlaylistSongs) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.cache.ServeHTTP(w, r)
}

// Changed returns neighbors list update event chan.
func (a *PlaylistSongs) Changed() <-chan struct{} {
	return a.changed
}

// Close closes update event chan.
func (a *PlaylistSongs) Close() {
	a.cache.Close()
}
