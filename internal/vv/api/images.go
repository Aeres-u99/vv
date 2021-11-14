package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/meiraka/vv/internal/songs"
)

const (
	pathAPIMusicImages = "/api/music/images"
)

type httpImages struct {
	Updating bool `json:"updating"`
}

type ImagesHandler struct {
	cache    *cache
	imgBatch *imgBatch
	mu       sync.RWMutex
	library  []map[string][]string
	update   chan bool
}

func NewImagesHandler(img []ImageProvider) (*ImagesHandler, error) {
	c, err := newCache(map[string]*httpStorage{})
	if err != nil {
		return nil, err
	}
	ret := &ImagesHandler{
		cache:    c,
		imgBatch: newImgBatch(img),
		update:   make(chan bool, 10),
	}
	go func() {
		for e := range ret.imgBatch.Event() {
			ret.cache.SetIfModified(&httpImages{Updating: e})
			ret.update <- e
		}
		close(ret.update)
	}()
	return ret, nil
}

func (a *ImagesHandler) ConvSong(s map[string][]string) (map[string][]string, bool) {
	// TODO: move to another function
	s = songs.AddTags(s)
	delete(s, "cover")
	cover, updated := a.imgBatch.GetURLs(s)
	if len(cover) != 0 {
		s["cover"] = cover
	}
	return s, updated
}

func (a *ImagesHandler) ConvSongs(s []map[string][]string) []map[string][]string {
	ret := make([]map[string][]string, len(s))
	needUpdates := make([]map[string][]string, 0, len(s))
	for i := range s {
		song, ok := a.ConvSong(s[i])
		ret[i] = song
		if !ok {
			needUpdates = append(needUpdates, song)
		}
	}
	if len(needUpdates) != 0 {
		a.imgBatch.Update(needUpdates)
	}
	return ret
}

func (a *ImagesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		a.cache.ServeHTTP(w, r)
		return
	}
	var req httpImages
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHTTPError(w, http.StatusBadRequest, err)
		return
	}
	if !req.Updating {
		writeHTTPError(w, http.StatusBadRequest, errors.New("requires updating=true"))
		return
	}
	a.mu.RLock()
	library := a.library
	a.mu.Unlock()
	a.imgBatch.Rescan(library)
	now := time.Now().UTC()
	r.Method = http.MethodGet
	a.cache.ServeHTTP(w, setUpdateTime(r, now))
}

// UpdateLibrarySongs set songs for rescan images.
func (a *ImagesHandler) UpdateLibrarySongs(songs []map[string][]string) {
	a.mu.Lock()
	a.library = songs
	a.mu.Unlock()
}

// Event returns batch state event chan.
func (a *ImagesHandler) Event() <-chan bool {
	return a.imgBatch.Event()
}

// Changed returns response body changes event chan.
func (a *ImagesHandler) Changed() <-chan struct{} {
	return a.cache.Changed()
}

// Close closes update event chan.
func (a *ImagesHandler) Close() {
	a.cache.Close()
}

// Shutdown stops background image updater api.
func (a *ImagesHandler) Shutdown(ctx context.Context) error {
	return a.imgBatch.Shutdown(ctx)
}
