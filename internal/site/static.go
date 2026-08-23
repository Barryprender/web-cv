package site

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// staticPrefix is where the embedded assets are mounted.
const staticPrefix = "/static/"

// contentTypes is declared explicitly rather than left to mime.TypeByExtension,
// whose answers on Windows come from the registry and are routinely wrong for
// .js and .css — a mislabelled module is refused outright under nosniff.
var contentTypes = map[string]string{
	".css":   "text/css; charset=utf-8",
	".js":    "text/javascript; charset=utf-8",
	".svg":   "image/svg+xml",
	".ico":   "image/x-icon",
	".png":   "image/png",
	".pdf":   "application/pdf",
	".woff2": "font/woff2",
	".txt":   "text/plain; charset=utf-8",
	".xml":   "application/xml",
}

// staticAsset is one embedded file, prepared once at startup.
type staticAsset struct {
	body        []byte
	contentType string
	etag        string
	version     string // short content hash, used for ?v= cache busting
}

// staticHandler serves the embedded assets with real cache validators.
//
// http.FileServer is deliberately not used: files read from an embed.FS carry a
// zero ModTime, so it emits neither Last-Modified nor ETag, and every asset is
// then refetched in full on every single navigation.
type staticHandler struct {
	assets map[string]staticAsset
}

func newStaticHandler(fsys fs.FS) (*staticHandler, error) {
	h := &staticHandler{assets: make(map[string]staticAsset)}

	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		body, readErr := fs.ReadFile(fsys, p)
		if readErr != nil {
			return readErr
		}
		sum := sha256.Sum256(body)
		digest := hex.EncodeToString(sum[:])

		contentType, ok := contentTypes[strings.ToLower(path.Ext(p))]
		if !ok {
			contentType = "application/octet-stream"
		}
		h.assets[p] = staticAsset{
			body:        body,
			contentType: contentType,
			etag:        `"` + digest[:16] + `"`,
			version:     digest[:8],
		}
		return nil
	})
	return h, err
}

func (h *staticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(path.Clean(r.URL.Path), staticPrefix)
	asset, ok := h.assets[name]
	if !ok {
		http.NotFound(w, r)
		return
	}

	header := w.Header()
	header.Set("Content-Type", asset.contentType)
	header.Set("ETag", asset.etag)

	// Cache forever only when the URL pins a specific version of the bytes:
	// either an explicit ?v= that still matches, or a filename that already
	// carries a content hash (the fonts). Everything else must revalidate,
	// which the ETag makes a cheap 304 rather than a full refetch.
	if r.URL.Query().Get("v") == asset.version || strings.HasPrefix(name, "fonts/") {
		header.Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		header.Set("Cache-Control", "public, no-cache")
	}

	// A zero modtime keeps ServeContent from emitting Last-Modified; the ETag
	// above is what drives conditional requests.
	http.ServeContent(w, r, path.Base(name), time.Time{}, bytes.NewReader(asset.body))
}

// assetURL returns the public URL for an embedded asset with its content hash
// attached, so a changed file is fetched immediately while an unchanged one
// stays in cache for a year. Templates reach this through the "asset" function.
//
// An unknown name is returned without a version rather than failing the render:
// it then shows up as an obvious 404 instead of silently dropping the tag.
func (h *staticHandler) assetURL(name string) string {
	name = strings.TrimPrefix(name, "/")
	asset, ok := h.assets[name]
	if !ok {
		return staticPrefix + name
	}
	return staticPrefix + name + "?v=" + asset.version
}
