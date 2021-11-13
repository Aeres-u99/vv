package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/meiraka/vv/internal/mpd"
)

const (
	pathAPIMusicStorageNeighbors = "/api/music/storage/neighbors"
)

// Neighbors provides neighbor storage name and uri.
type Neighbors struct {
	mpd interface {
		ListNeighbors(context.Context) ([]map[string]string, error)
	}
	cache *cache
}

// NewNeighbors initilize Neighbors cache with mpd connection.
func NewNeighbors(mpd interface {
	ListNeighbors(context.Context) ([]map[string]string, error)
}) (*Neighbors, error) {
	c, err := newCache(map[string]*httpStorage{})
	if err != nil {
		return nil, err
	}
	return &Neighbors{
		mpd:   mpd,
		cache: c,
	}, nil
}

// Update updates neighbors list.
func (a *Neighbors) Update(ctx context.Context) error {
	ret := map[string]*httpStorage{}
	ms, err := a.mpd.ListNeighbors(ctx)
	if err != nil {
		// skip command error to support old mpd
		var perr *mpd.CommandError
		if errors.As(err, &perr) {
			a.cache.SetIfModified(ret)
			return nil
		}
		return err
	}
	for _, m := range ms {
		ret[m["name"]] = &httpStorage{
			URI: stringPtr(m["neighbor"]),
		}
	}
	a.cache.SetIfModified(ret)
	return nil
}

// ServeHTTP responses neighbors list as json format.
func (a *Neighbors) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.cache.ServeHTTP(w, r)
}

// Changed returns neighbors list update event chan.
func (a *Neighbors) Changed() <-chan struct{} {
	return a.cache.Changed()
}

// Close closes update event chan.
func (a *Neighbors) Close() {
	a.cache.Close()
}
