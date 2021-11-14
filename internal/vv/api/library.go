package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

const pathAPIMusicLibrary = "/api/music/library"

type httpLibraryInfo struct {
	Updating bool `json:"updating"`
}

type MPDLibraryAPI interface {
	Update(context.Context, string) (map[string]string, error)
}

type Library struct {
	mpd   MPDLibraryAPI
	cache *cache
}

func NewLibrary(mpd MPDLibraryAPI) (*Library, error) {
	c, err := newCache(&httpLibraryInfo{})
	if err != nil {
		return nil, err
	}
	return &Library{
		mpd:   mpd,
		cache: c,
	}, nil
}

func (a *Library) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		a.cache.ServeHTTP(w, r)
		return
	}
	var req httpLibraryInfo
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHTTPError(w, http.StatusBadRequest, err)
		return
	}
	if !req.Updating {
		writeHTTPError(w, http.StatusBadRequest, errors.New("requires updating=true"))
		return
	}
	ctx := r.Context()
	now := time.Now().UTC()
	if _, err := a.mpd.Update(ctx, ""); err != nil {
		writeHTTPError(w, http.StatusInternalServerError, err)
		return
	}
	r.Method = http.MethodGet
	a.cache.ServeHTTP(w, setUpdateTime(r, now))
}

func (a *Library) Update(ctx context.Context, updating bool) error {
	_, err := a.cache.SetIfModified(&httpLibraryInfo{Updating: updating})
	return err
}
