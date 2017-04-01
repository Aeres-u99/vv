package main

import (
	"encoding/json"
	"github.com/fhs/gompd/mpd"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLibrary(t *testing.T) {
	m := new(MockMusic)
	api := apiHandler{m}
	ts := httptest.NewServer(http.HandlerFunc(api.library))
	defer ts.Close()
	t.Run("no parameter", func(t *testing.T) {
		m.LibraryRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.LibraryRet2 = time.Unix(0, 0)
		res, err := http.Get(ts.URL)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		st := struct {
			Data   []mpd.Attrs `json:"data"`
			Errors error       `json:"errors"`
		}{[]mpd.Attrs{}, nil}
		json.Unmarshal(body, &st)
		if !reflect.DeepEqual(m.LibraryRet1, st.Data) {
			t.Errorf("unexpected body: %s", body)
		}
		if st.Errors != nil {
			t.Errorf("unexpected body: %s", body)
		}
	})
	t.Run("If-Modified-Since", func(t *testing.T) {
		m.LibraryRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.LibraryRet2 = time.Unix(60, 0)
		req, _ := http.NewRequest("GET", ts.URL, nil)
		req.Header.Set("If-Modified-Since", m.LibraryRet2.Format(http.TimeFormat))
		client := new(http.Client)
		res, err := client.Do(req)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 304 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
}
func TestPlaylist(t *testing.T) {
	m := new(MockMusic)
	api := apiHandler{m}
	ts := httptest.NewServer(http.HandlerFunc(api.playlist))
	defer ts.Close()
	t.Run("no parameter", func(t *testing.T) {
		m.PlaylistRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.PlaylistRet2 = time.Unix(0, 0)
		res, err := http.Get(ts.URL)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		st := struct {
			Data   []mpd.Attrs `json:"data"`
			Errors error       `json:"errors"`
		}{[]mpd.Attrs{}, nil}
		json.Unmarshal(body, &st)
		if !reflect.DeepEqual(m.PlaylistRet1, st.Data) {
			t.Errorf("unexpected body: %s", body)
		}
		if st.Errors != nil {
			t.Errorf("unexpected body: %s", body)
		}
	})
	t.Run("If-Modified-Since", func(t *testing.T) {
		m.PlaylistRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.PlaylistRet2 = time.Unix(60, 0)
		req, _ := http.NewRequest("GET", ts.URL, nil)
		req.Header.Set("If-Modified-Since", m.PlaylistRet2.Format(http.TimeFormat))
		client := new(http.Client)
		res, err := client.Do(req)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 304 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
	t.Run("sort", func(t *testing.T) {
		m.SortPlaylistErr = nil
		j := strings.NewReader(
			"{\"action\": \"sort\", \"keys\": [\"file\"], \"uri\": \"path\"}",
		)
		res, err := http.Post(ts.URL, "application/json", j)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}

		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		b := struct {
			Errors error `json:"errors"`
		}{nil}
		json.Unmarshal(body, &b)
		if b.Errors != nil {
			t.Errorf("unexpected body: %s", body)
		}
	})
}
func TestOutput(t *testing.T) {
	m := new(MockMusic)
	setHandle(m)
	ts := httptest.NewServer(http.DefaultServeMux)
	defer ts.Close()
	t.Run("no parameter", func(t *testing.T) {
		m.OutputsRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.OutputsRet2 = time.Unix(0, 0)
		res, err := http.Get(ts.URL + "/api/outputs")
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		st := struct {
			Data   []mpd.Attrs `json:"data"`
			Errors error       `json:"errors"`
		}{[]mpd.Attrs{}, nil}
		json.Unmarshal(body, &st)
		if !reflect.DeepEqual(m.OutputsRet1, st.Data) {
			t.Errorf("unexpected body: %s", body)
		}
		if st.Errors != nil {
			t.Errorf("unexpected body: %s", body)
		}
	})
	t.Run("If-Modified-Since", func(t *testing.T) {
		m.OutputsRet1 = []mpd.Attrs{mpd.Attrs{"foo": "bar"}}
		m.OutputsRet2 = time.Unix(60, 0)
		req, _ := http.NewRequest("GET", ts.URL+"/api/outputs", nil)
		req.Header.Set("If-Modified-Since", m.OutputsRet2.Format(http.TimeFormat))
		client := new(http.Client)
		res, err := client.Do(req)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 304 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
	t.Run("enable", func(t *testing.T) {
		j := strings.NewReader(
			"{\"outputenabled\": true}",
		)
		res, err := http.Post(ts.URL+"/api/outputs/1", "application/json", j)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		if m.OutputArg1 != 1 || m.OutputArg2 != true {
			t.Errorf("unexpected arguments: %d, %t", m.OutputArg1, m.OutputArg2)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		b := struct {
			Errors error `json:"errors"`
		}{nil}
		json.Unmarshal(body, &b)
		if b.Errors != nil {
			t.Errorf("unexpected body: %s", body)
		}
	})
}
func TestCurrent(t *testing.T) {
	m := new(MockMusic)
	api := apiHandler{m}
	ts := httptest.NewServer(http.HandlerFunc(api.current))
	defer ts.Close()
	t.Run("no parameter", func(t *testing.T) {
		m.CurrentRet1 = mpd.Attrs{"foo": "bar"}
		m.CurrentRet2 = time.Unix(0, 0)
		res, err := http.Get(ts.URL)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		st := struct {
			Data   mpd.Attrs `json:"data"`
			Errors error     `json:"errors"`
		}{mpd.Attrs{}, nil}
		json.Unmarshal(body, &st)
		if !reflect.DeepEqual(m.CurrentRet1, st.Data) {
			t.Errorf("unexpected body: %s", body)
		}
		if st.Errors != nil {
			t.Errorf("unexpected body: %s", body)
		}
	})
	t.Run("If-Modified-Since", func(t *testing.T) {
		m.CurrentRet1 = mpd.Attrs{"foo": "bar"}
		m.CurrentRet2 = time.Unix(60, 0)
		req, _ := http.NewRequest("GET", ts.URL, nil)
		req.Header.Set("If-Modified-Since", m.CurrentRet2.Format(http.TimeFormat))
		client := new(http.Client)
		res, err := client.Do(req)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 304 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
}
func TestControl(t *testing.T) {
	m := new(MockMusic)
	api := apiHandler{m}
	ts := httptest.NewServer(http.HandlerFunc(api.control))
	defer ts.Close()
	t.Run("no parameter", func(t *testing.T) {
		s := convStatus(mpd.Attrs{}, 0)
		m.StatusRet1 = s
		m.StatusRet2 = time.Unix(0, 0)
		res, err := http.Get(ts.URL)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		st := struct {
			Data   PlayerStatus `json:"data"`
			Errors error        `json:"errors"`
		}{PlayerStatus{}, nil}
		json.Unmarshal(body, &st)
		if !reflect.DeepEqual(s, st.Data) {
			t.Errorf("unexpected body: %s", body)
		}
		if st.Errors != nil {
			t.Errorf("unexpected body: %s", body)
		}
	})
	t.Run("If-Modified-Since", func(t *testing.T) {
		s := convStatus(mpd.Attrs{}, 60)
		m.StatusRet1 = s
		m.StatusRet2 = time.Unix(60, 0)
		req, _ := http.NewRequest("GET", ts.URL, nil)
		req.Header.Set("If-Modified-Since", m.StatusRet2.Format(http.TimeFormat))
		client := new(http.Client)
		res, err := client.Do(req)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 304 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
	t.Run("action=play", func(t *testing.T) {
		res, err := http.Get(ts.URL + "?action=play")
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})

	t.Run("action=pause", func(t *testing.T) {
		res, err := http.Get(ts.URL + "?action=pause")
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})

	t.Run("action=next", func(t *testing.T) {
		res, err := http.Get(ts.URL + "?action=next")
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})

	t.Run("action=prev", func(t *testing.T) {
		res, err := http.Get(ts.URL + "?action=prev")
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
	})
	t.Run("volume", func(t *testing.T) {
		j := strings.NewReader(
			"{\"volume\": 1}",
		)
		res, err := http.Post(ts.URL, "application/json", j)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		b := struct {
			Errors error `json:"errors"`
		}{nil}
		json.Unmarshal(body, &b)
		if b.Errors != nil {
			t.Errorf("unexpected body: %s", body)
		}
	})
	t.Run("repeat", func(t *testing.T) {
		j := strings.NewReader(
			"{\"repeat\": true}",
		)
		res, err := http.Post(ts.URL, "application/json", j)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		b := struct {
			Errors error `json:"errors"`
		}{nil}
		json.Unmarshal(body, &b)
		if b.Errors != nil {
			t.Errorf("unexpected body: %s", body)
		}
	})
	t.Run("random", func(t *testing.T) {
		j := strings.NewReader(
			"{\"random\": true}",
		)
		res, err := http.Post(ts.URL, "application/json", j)
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		}
		if res.StatusCode != 200 {
			t.Errorf("unexpected status %d", res.StatusCode)
		}
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		b := struct {
			Errors error `json:"errors"`
		}{nil}
		json.Unmarshal(body, &b)
		if b.Errors != nil {
			t.Errorf("unexpected body: %s", body)
		}
	})
}

type MockMusic struct {
	PlayErr          error
	PauseErr         error
	NextErr          error
	PrevErr          error
	VolumeArg1       int
	VolumeErr        error
	RepeatArg1       bool
	RepeatErr        error
	RandomArg1       bool
	RandomErr        error
	PlaylistRet1     []mpd.Attrs
	PlaylistRet2     time.Time
	LibraryRet1      []mpd.Attrs
	LibraryRet2      time.Time
	OutputsRet1      []mpd.Attrs
	OutputsRet2      time.Time
	OutputArg1       int
	OutputArg2       bool
	OutputRet1       error
	CurrentRet1      mpd.Attrs
	CurrentRet2      time.Time
	CommentsRet1     mpd.Attrs
	CommentsRet2     time.Time
	StatusRet1       PlayerStatus
	StatusRet2       time.Time
	SortPlaylistArg1 []string
	SortPlaylistArg2 string
	SortPlaylistErr  error
}

func (p *MockMusic) Play() error {
	return p.PlayErr
}

func (p *MockMusic) Pause() error {
	return p.PauseErr
}
func (p *MockMusic) Next() error {
	return p.NextErr
}
func (p *MockMusic) Prev() error {
	return p.PrevErr
}
func (p *MockMusic) Volume(i int) error {
	p.VolumeArg1 = i
	return p.VolumeErr
}
func (p *MockMusic) Repeat(b bool) error {
	p.RepeatArg1 = b
	return p.RepeatErr
}
func (p *MockMusic) Random(b bool) error {
	p.RandomArg1 = b
	return p.RandomErr
}
func (p *MockMusic) Comments() (mpd.Attrs, time.Time) {
	return p.CommentsRet1, p.CommentsRet2
}
func (p *MockMusic) Current() (mpd.Attrs, time.Time) {
	return p.CurrentRet1, p.CurrentRet2
}
func (p *MockMusic) Library() ([]mpd.Attrs, time.Time) {
	return p.LibraryRet1, p.LibraryRet2
}
func (p *MockMusic) Outputs() ([]mpd.Attrs, time.Time) {
	return p.OutputsRet1, p.OutputsRet2
}
func (p *MockMusic) Output(id int, on bool) error {
	p.OutputArg1, p.OutputArg2 = id, on
	return p.OutputRet1
}
func (p *MockMusic) Playlist() ([]mpd.Attrs, time.Time) {
	return p.PlaylistRet1, p.PlaylistRet2
}
func (p *MockMusic) Status() (PlayerStatus, time.Time) {
	return p.StatusRet1, p.StatusRet2
}
func (p *MockMusic) SortPlaylist(s []string, u string) error {
	p.SortPlaylistArg1 = s
	p.SortPlaylistArg2 = u
	return p.SortPlaylistErr
}
