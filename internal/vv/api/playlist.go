package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/songs"
)

type httpPlaylistInfo struct {
	// current track
	Current *int `json:"current,omitempty"`
	// sort functions
	Sort    []string     `json:"sort,omitempty"`
	Filters [][2]*string `json:"filters,omitempty"`
	Must    int          `json:"must,omitempty"`
}

// Playlist provides current playlist sort function.
type Playlist struct {
	mpd         MPDPlaylistAPI
	library     []map[string][]string
	librarySort []map[string][]string
	playlist    []map[string][]string
	cache       *cache
	data        *httpPlaylistInfo
	mu          sync.RWMutex
	sem         chan struct{}
	config      *Config
}

type MPDPlaylistAPI interface {
	Play(context.Context, int) error
	ExecCommandList(context.Context, *mpd.CommandList) error
}

func NewPlaylist(mpd MPDPlaylistAPI, config *Config) (*Playlist, error) {
	c, err := newCache(&httpPlaylistInfo{})
	if err != nil {
		return nil, err
	}
	sem := make(chan struct{}, 1)
	sem <- struct{}{}
	return &Playlist{
		mpd:    mpd,
		cache:  c,
		data:   &httpPlaylistInfo{},
		sem:    sem,
		config: config,
	}, nil
}

func (a *Playlist) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		a.cache.ServeHTTP(w, r)
		return
	}
	var req httpPlaylistInfo
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHTTPError(w, http.StatusBadRequest, err)
		return
	}

	if req.Current == nil || req.Filters == nil || req.Sort == nil {
		writeHTTPError(w, http.StatusBadRequest, errors.New("current, filters and sort fields are required"))
		return
	}

	select {
	case <-a.sem:
	default:
		// TODO: switch to better status code
		writeHTTPError(w, http.StatusServiceUnavailable, errors.New("updating playlist"))
		return
	}
	defer func() { a.sem <- struct{}{} }()

	a.mu.Lock()
	librarySort, filters, newpos := songs.WeakFilterSort(a.library, req.Sort, req.Filters, req.Must, 9999, *req.Current)
	a.librarySort = librarySort
	update := !songs.SortEqual(a.playlist, a.librarySort)
	a.mu.Unlock()
	cl := &mpd.CommandList{}
	cl.Clear()
	for i := range a.librarySort {
		cl.Add(a.librarySort[i]["file"][0])
	}
	cl.Play(newpos)
	if !update {
		a.updateSort(req.Sort, filters, req.Must)
		now := time.Now().UTC()
		ctx := r.Context()
		if err := a.mpd.Play(ctx, newpos); err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		r.Method = http.MethodGet
		a.cache.ServeHTTP(w, setUpdateTime(r, now))
		return
	}
	r.Method = http.MethodGet
	a.cache.ServeHTTP(w, setUpdateTime(r, time.Now().UTC()))
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), a.config.BackgroundTimeout)
		defer cancel()
		select {
		case <-a.sem:
		case <-ctx.Done():
			return
		}
		defer func() { a.sem <- struct{}{} }()
		if err := a.mpd.ExecCommandList(ctx, cl); err != nil {
			return
		}
		a.updateSort(req.Sort, filters, req.Must)
	}()
}

func (a *Playlist) UpdateCurrent(pos int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	data := &httpPlaylistInfo{
		Current: &pos,
		Sort:    a.data.Sort,
		Filters: a.data.Filters,
		Must:    a.data.Must,
	}
	_, err := a.cache.SetIfModified(data)
	if err != nil {
		return err
	}
	a.data = data
	return nil
}

func (a *Playlist) updateSort(sort []string, filters [][2]*string, must int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	data := &httpPlaylistInfo{
		Current: a.data.Current,
		Sort:    sort,
		Filters: filters,
		Must:    must,
	}
	_, err := a.cache.SetIfModified(data)
	if err != nil {
		return err
	}
	a.data = data
	return nil
}

func (a *Playlist) UpdatePlaylistSongs(i []map[string][]string) {
	a.mu.Lock()
	a.playlist = i
	unsort := a.data.Sort != nil && !songs.SortEqual(a.playlist, a.librarySort)
	a.mu.Unlock()
	if unsort {
		a.updateSort(nil, nil, 0)
	}
}

func (a *Playlist) UpdateLibrarySongs(i []map[string][]string) {
	a.mu.Lock()
	a.library = i
	a.librarySort = nil
	a.mu.Unlock()
}
