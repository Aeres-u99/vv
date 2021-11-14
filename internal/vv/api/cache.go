package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/meiraka/vv/internal/gzip"
	"github.com/meiraka/vv/internal/request"
)

type cache struct {
	changed chan struct{}
	json    []byte
	gzjson  []byte
	date    time.Time
	mu      sync.RWMutex
}

func newCache(i interface{}) (*cache, error) {
	b, gz, err := cacheBinary(i)
	if err != nil {
		return nil, err
	}
	c := &cache{
		changed: make(chan struct{}, 1),
		json:    b,
		gzjson:  gz,
		date:    time.Now().UTC(),
	}
	return c, nil
}

func (c *cache) Close() {
	c.mu.Lock()
	close(c.changed)
	c.mu.Unlock()
}

func (c *cache) Changed() <-chan struct{} {
	return c.changed
}

func (c *cache) Set(i interface{}) error {
	_, err := c.set(i, true)
	return err
}

func (c *cache) SetIfModified(i interface{}) (bool, error) {
	return c.set(i, false)
}

func (c *cache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.mu.RLock()
	b, gz, date := c.json, c.gzjson, c.date
	c.mu.RUnlock()
	etag := fmt.Sprintf(`"%d.%d"`, date.Unix(), date.Nanosecond())
	if request.NoneMatch(r, etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	if !request.ModifiedSince(r, date) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Add("Cache-Control", "max-age=0")
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.Header().Add("Last-Modified", date.Format(http.TimeFormat))
	w.Header().Add("Vary", "Accept-Encoding")
	w.Header().Add("ETag", etag)
	status := http.StatusOK
	if getUpdateTime(r).After(date) {
		status = http.StatusAccepted
	}
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && gz != nil {
		w.Header().Add("Content-Encoding", "gzip")
		w.Header().Add("Content-Length", strconv.Itoa(len(gz)))
		w.WriteHeader(status)
		w.Write(gz)
		return
	}
	w.Header().Add("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(status)
	w.Write(b)
}

func (c *cache) set(i interface{}, force bool) (bool, error) {
	n, gz, err := cacheBinary(i)
	if err != nil {
		return false, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	o := c.json
	if force || !bytes.Equal(o, n) {
		c.json = n
		c.date = time.Now().UTC()
		c.gzjson = gz
		select {
		case c.changed <- struct{}{}:
		default:
		}
		return true, nil
	}
	return false, nil
}

func cacheBinary(i interface{}) ([]byte, []byte, error) {
	n, err := json.Marshal(i)
	if err != nil {
		return nil, nil, err
	}
	gz, err := gzip.Encode(n)
	if err != nil {
		return n, nil, nil
	}
	return n, gz, nil
}

type jsonCache struct {
	event  chan string
	index  []string
	data   [][]byte
	gzdata [][]byte
	date   []time.Time
	mu     sync.RWMutex
}

func newJSONCache() *jsonCache {
	return &jsonCache{
		event:  make(chan string, 100),
		index:  []string{},
		data:   [][]byte{},
		gzdata: [][]byte{},
		date:   []time.Time{},
	}
}

func (c *jsonCache) Close() {
	c.mu.Lock()
	close(c.event)
	c.mu.Unlock()
}

func (c *jsonCache) Event() <-chan string {
	return c.event
}

func (c *jsonCache) Set(path string, i interface{}) error {
	return c.set(path, i, true)
}

func (c *jsonCache) SetIfModified(path string, i interface{}) error {
	return c.set(path, i, false)
}

func (c *jsonCache) SetIfNone(path string, i interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	pos := c.getPos(path)
	if c.data[pos] != nil {
		return nil
	}
	n, err := json.Marshal(i)
	if err != nil {
		return err
	}

	c.data[pos] = n
	c.date[pos] = time.Now().UTC()
	gz, err := gzip.Encode(n)
	if err == nil {
		c.gzdata[pos] = gz
	} else {
		c.gzdata[pos] = nil
	}
	select {
	case c.event <- path:
	default:
	}
	return nil
}

func (c *jsonCache) getPos(path string) int {
	for i := range c.index {
		if c.index[i] == path {
			return i
		}
	}
	c.index = append(c.index, path)
	c.data = append(c.data, nil)
	c.gzdata = append(c.gzdata, nil)
	c.date = append(c.date, time.Time{})
	return len(c.index) - 1
}

func (c *jsonCache) set(path string, i interface{}, force bool) error {
	n, err := json.Marshal(i)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	pos := c.getPos(path)

	o := c.data[pos]
	if force || !bytes.Equal(o, n) {
		c.data[pos] = n
		c.date[pos] = time.Now().UTC()
		gz, err := gzip.Encode(n)
		if err == nil {
			c.gzdata[pos] = gz
		} else {
			c.gzdata[pos] = nil
		}
		select {
		case c.event <- path:
		default:
		}
	}
	return nil
}

func (c *jsonCache) Get(path string) (data, gzdata []byte, l time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for i := range c.index {
		if c.index[i] == path {
			return c.data[i], c.gzdata[i], c.date[i]
		}
	}
	return nil, nil, time.Time{}
}

func (c *jsonCache) Handler(path string) http.HandlerFunc {
	c.mu.Lock()
	pos := c.getPos(path)
	c.mu.Unlock()
	return func(w http.ResponseWriter, r *http.Request) {
		c.mu.RLock()
		b, gz, date := c.data[pos], c.gzdata[pos], c.date[pos]
		c.mu.RUnlock()
		etag := fmt.Sprintf(`"%d.%d"`, date.Unix(), date.Nanosecond())
		if request.NoneMatch(r, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if !request.ModifiedSince(r, date) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Add("Cache-Control", "max-age=0")
		w.Header().Add("Content-Type", "application/json; charset=utf-8")
		w.Header().Add("Last-Modified", date.Format(http.TimeFormat))
		w.Header().Add("Vary", "Accept-Encoding")
		w.Header().Add("ETag", etag)
		status := http.StatusOK
		if getUpdateTime(r).After(date) {
			status = http.StatusAccepted
		}
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && gz != nil {
			w.Header().Add("Content-Encoding", "gzip")
			w.Header().Add("Content-Length", strconv.Itoa(len(gz)))
			w.WriteHeader(status)
			w.Write(gz)
			return
		}
		w.Header().Add("Content-Length", strconv.Itoa(len(b)))
		w.WriteHeader(status)
		w.Write(b)
	}
}
