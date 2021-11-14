package api

import (
	"context"
	"net/http"
	"sync"
)

type MPDLibrarySongsAPI interface {
	ListAllInfo(context.Context, string) ([]map[string][]string, error)
}

func NewLibrarySongs(mpd MPDLibrarySongsAPI, songsHook func([]map[string][]string) []map[string][]string, eventHooks ...func([]map[string][]string)) (*LibrarySongs, error) {
	cache, err := newCache([]map[string][]string{})
	if err != nil {
		return nil, err
	}
	return &LibrarySongs{
		mpd:        mpd,
		cache:      cache,
		songsHook:  songsHook,
		eventHooks: eventHooks,
	}, nil

}

type LibrarySongs struct {
	mpd        MPDLibrarySongsAPI
	cache      *cache
	songsHook  func([]map[string][]string) []map[string][]string
	eventHooks []func([]map[string][]string)
	data       []map[string][]string
	mu         sync.Mutex
}

func (a *LibrarySongs) Update(ctx context.Context) error {
	l, err := a.mpd.ListAllInfo(ctx, "/")
	if err != nil {
		return err
	}
	v := a.songsHook(l)
	// force update to skip []byte compare
	if err := a.cache.Set(v); err != nil {
		return err
	}
	a.mu.Lock()
	a.data = v
	a.mu.Unlock()
	for i := range a.eventHooks {
		a.eventHooks[i](v)
	}
	return nil
}

// ServeHTTP responses library song list as json format.
func (a *LibrarySongs) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.cache.ServeHTTP(w, r)
}

// Changed returns library song list update event chan.
func (a *LibrarySongs) Changed() <-chan struct{} {
	return a.cache.Changed()
}

// Close closes update event chan.
func (a *LibrarySongs) Close() {
	a.cache.Close()
}
