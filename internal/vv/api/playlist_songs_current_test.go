package api_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/meiraka/vv/internal/vv/api"
)

func TestGetPlaylistSongsCurrent(t *testing.T) {
	songHook, randValue := testSongHook()
	for label, tt := range map[string][]struct {
		currentSong func() (map[string][]string, error)
		err         error
		want        string
		cache       map[string][]string
		changed     bool
	}{
		"ok": {{
			currentSong: func() (map[string][]string, error) { return map[string][]string{"file": {"/foo/bar.mp3"}}, nil },
			want:        fmt.Sprintf(`{"%s":["%s"],"file":["/foo/bar.mp3"]}`, randValue, randValue),
			cache:       map[string][]string{"file": {"/foo/bar.mp3"}, randValue: {randValue}},
			changed:     true,
		}},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdPlaylistSongsCurrent{}
			h, err := api.NewPlaylistSongsCurrentHandler(mpd, songHook)
			if err != nil {
				t.Fatalf("api.NewPlaylistSongsCurrentHandler() = %v, %v", h, err)
			}
			for i := range tt {
				t.Run(fmt.Sprint(i), func(t *testing.T) {
					mpd.t = t
					mpd.currentSong = tt[i].currentSong
					if err := h.Update(context.TODO()); !errors.Is(err, tt[i].err) {
						t.Errorf("handler.Update(context.TODO()) = %v; want %v", err, tt[i].err)
					}
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					w := httptest.NewRecorder()
					h.ServeHTTP(w, r)
					if got := w.Body.String(); got != tt[i].want {
						t.Errorf("ServeHTTP response: got\n%s; want\n%s", got, tt[i].want)
					}
					if got, want := w.Result().StatusCode, http.StatusOK; got != want {
						t.Errorf("ServeHTTP response status: got %d; want %d", got, want)
					}
					changed := false
					select {
					case <-h.Changed():
						changed = true
					default:
					}
					if changed != tt[i].changed {
						t.Errorf("changed = %v; want %v", changed, tt[i].changed)
					}
				})
			}
		})
	}
}

type mpdPlaylistSongsCurrent struct {
	t           *testing.T
	currentSong func() (map[string][]string, error)
}

func (m *mpdPlaylistSongsCurrent) CurrentSong(context.Context) (map[string][]string, error) {
	m.t.Helper()
	if m.currentSong == nil {
		m.t.Fatal("no CurrentSong mock function")
	}
	return m.currentSong()
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func testSongHook() (func(s map[string][]string) map[string][]string, string) {
	key := fmt.Sprint(rand.Int())
	return func(s map[string][]string) map[string][]string {
		s[key] = []string{key}
		return s
	}, key
}
