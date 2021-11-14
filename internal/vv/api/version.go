package api

import (
	"fmt"
	"net/http"
	"runtime"
)

const (
	pathAPIVersion = "/api/version"
)

var goVersion = fmt.Sprintf("%s %s %s", runtime.Version(), runtime.GOOS, runtime.GOARCH)

type httpVersion struct {
	App string `json:"app"`
	Go  string `json:"go"`
	MPD string `json:"mpd"`
}

type Version struct {
	mpd     MPDVersionAPI
	cache   *cache
	version string
}

// MPDVersionAPI represents mpd api for Version API.
type MPDVersionAPI interface {
	Version() string
}

func NewVersion(mpd MPDVersionAPI, version string) (*Version, error) {
	c, err := newCache(map[string]*httpVersion{})
	if err != nil {
		return nil, err
	}
	return &Version{
		mpd:     mpd,
		cache:   c,
		version: version,
	}, nil
}

func (a *Version) Update() error {
	mpdVersion := a.mpd.Version()
	if len(mpdVersion) == 0 {
		mpdVersion = "unknown"
	}
	_, err := a.cache.SetIfModified(&httpVersion{App: a.version, Go: goVersion, MPD: mpdVersion})
	return err
}

func (a *Version) UpdateNoMPD() error {
	_, err := a.cache.SetIfModified(&httpVersion{App: a.version, Go: goVersion})
	return err
}

// ServeHTTP responses version as json format.
func (a *Version) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.cache.ServeHTTP(w, r)
}

// Changed returns version update event chan.
func (a *Version) Changed() <-chan struct{} {
	return a.cache.Changed()
}

// Close closes update event chan.
func (a *Version) Close() {
	a.cache.Close()
}
