package vv

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/meiraka/vv/internal/vv/assets"
)

//go:embed app.html
var appHTML []byte

// HTMLConfig is options for HTMLHandler.
type HTMLConfig struct {
	Tree      Tree     // playlist view definition.
	TreeOrder []string // order of playlist tree.
}

// NewHTMLHander creates http.Handler for app root html.
func NewHTMLHander(config *HTMLConfig) (http.Handler, error) {
	c := new(HTMLConfig)
	if config != nil {
		*c = *config
	}
	if c.Tree == nil && c.TreeOrder == nil {
		c.Tree = DefaultTree
		c.TreeOrder = DefaultTreeOrder
	}
	if c.Tree == nil && c.TreeOrder != nil {
		return nil, errors.New("invalid config: no tree")
	}
	if c.Tree != nil && c.TreeOrder == nil {
		return nil, errors.New("invalid config: no tree order")
	}
	extra := map[string]string{
		"AssetsAppCSSHash": string(assets.AppCSSHash),
		"AssetsAppJSHash":  string(assets.AppJSHash),
	}
	jsonTree, err := json.Marshal(c.Tree)
	if err != nil {
		return nil, fmt.Errorf("tree: %v", err)
	}
	extra["TREE"] = string(jsonTree)
	jsonTreeOrder, err := json.Marshal(c.TreeOrder)
	if err != nil {
		return nil, fmt.Errorf("tree order: %v", err)
	}
	extra["TREE_ORDER"] = string(jsonTreeOrder)
	return i18nHandler(filepath.Join("assets", "app.html"), appHTML, extra)
}
