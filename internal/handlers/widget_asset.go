package handlers

import (
	"io/fs"
	"log"
	"net/http"
	"strconv"

	"github.com/Nezzy-joe/nez-widget-platform/internal/widgetassets"
)

// WidgetAssetHandler serves versioned JavaScript widget assets.
type WidgetAssetHandler struct{}

func NewWidgetAssetHandler() *WidgetAssetHandler {
	return &WidgetAssetHandler{}
}

func (h *WidgetAssetHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.URL.Path != "/widget/v1/widget.js" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	script, err := fs.ReadFile(widgetassets.Files, "widget/v1/widget.js")
	if err != nil {
		log.Printf("read embedded widget asset failed: %v", err)
		writeError(
			w,
			http.StatusInternalServerError,
			"widget asset unavailable",
		)
		return
	}

	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", stringLength(script))

	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodGet {
		_, _ = w.Write(script)
	}
}

// stringLength returns the decimal representation of a byte count.
func stringLength(data []byte) string {
	return strconv.Itoa(len(data))
}
