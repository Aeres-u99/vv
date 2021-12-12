package assets

import (
	"crypto/md5"
	_ "embed"
	"encoding/hex"
)

var (
	//go:embed app-black.png
	AppBlackPNG     []byte
	AppBlackPNGHash []byte = hash(AppBlackPNG)
	//go:embed app-black.svg
	AppBlackSVG     []byte
	AppBlackSVGHash []byte = hash(AppBlackSVG)
	//go:embed app.css
	AppCSS     []byte
	AppCSSHash []byte = hash(AppCSS)
	//go:embed app.js
	AppJS     []byte
	AppJSHash []byte = hash(AppJS)
	//go:embed app.png
	AppPNG     []byte
	AppPNGHash []byte = hash(AppPNG)
	//go:embed app.svg
	AppSVG     []byte
	AppSVGHash []byte = hash(AppSVG)
	//go:embed manifest.json
	ManifestJSON     []byte
	ManifestJSONHash []byte = hash(ManifestJSON)
	//go:embed nocover.svg
	NocoverSVG     []byte
	NocoverSVGHash []byte = hash(NocoverSVG)
	//go:embed w.png
	WPNG     []byte
	WPNGHash []byte = hash(WPNG)
)

func hash(b []byte) []byte {
	hasher := md5.New()
	hasher.Write(b)
	sum := hasher.Sum(nil)
	ret := make([]byte, hex.EncodedLen(len(sum)))
	hex.Encode(ret, sum)
	return ret
}
