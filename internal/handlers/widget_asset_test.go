package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWidgetAssetServesEmbeddedJavaScript(t *testing.T) {
	handler := NewWidgetAssetHandler()

	req := httptest.NewRequest(
		http.MethodGet,
		"/widget/v1/widget.js",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if got := rec.Header().Get("Content-Type"); got !=
		"application/javascript; charset=utf-8" {
		t.Fatalf("unexpected content type: %q", got)
	}

	if got := rec.Header().Get("Cache-Control"); got !=
		"public, max-age=300" {
		t.Fatalf("unexpected cache policy: %q", got)
	}

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("expected nosniff header")
	}

	body := rec.Body.String()
	for _, marker := range []string{
		"data-nez-widget",
		"/public/widgets/",
		"/submissions",
	} {
		if !strings.Contains(body, marker) {
			t.Errorf("served script does not contain expected marker %q", marker)
		}
	}
}

func TestWidgetAssetRejectsUnsupportedMethod(t *testing.T) {
	handler := NewWidgetAssetHandler()

	req := httptest.NewRequest(
		http.MethodPost,
		"/widget/v1/widget.js",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}

	if rec.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("unexpected Allow header: %q", rec.Header().Get("Allow"))
	}
}

func TestWidgetAssetHeadDoesNotReturnBody(t *testing.T) {
	handler := NewWidgetAssetHandler()

	req := httptest.NewRequest(
		http.MethodHead,
		"/widget/v1/widget.js",
		nil,
	)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD response should have no body, got %d bytes", rec.Body.Len())
	}
}
