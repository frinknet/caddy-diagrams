package diagram

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func TestHandlerMatches(t *testing.T) {
	for _, tt := range []struct {
		name string
		h    Handler
		path string
		want bool
	}{
		{"auto mmd", Handler{Type: "auto"}, "/a.mmd", true},
		{"auto d2", Handler{Type: "auto"}, "/a.d2", true},
		{"auto dot", Handler{Type: "auto"}, "/a.dot", true},
		{"auto html", Handler{Type: "auto"}, "/a.html", false},
		{"auto markdown", Handler{Type: "auto"}, "/a.md", false},
		{"markdown md", Handler{Type: "markdown"}, "/a.md", true},
		{"markdown html", Handler{Type: "markdown"}, "/a.html", false},
		{"css default", Handler{Type: "css"}, "/_diagrams/diagrams.css", true},
		{"css other", Handler{Type: "css"}, "/a.css", false},
		{"exact path", Handler{Type: "auto", Path: "/a.mmd"}, "/a.mmd", true},
		{"exact path miss", Handler{Type: "auto", Path: "/a.mmd"}, "/b.mmd", false},
		{"prefix path", Handler{Type: "auto", Path: "/docs/"}, "/docs/a.mmd", true},
		{"prefix path miss", Handler{Type: "auto", Path: "/docs/"}, "/a.mmd", false},
		{"glob path", Handler{Type: "auto", Path: "/docs/*.mmd"}, "/docs/a.mmd", true},
		{"glob path miss", Handler{Type: "auto", Path: "/docs/*.mmd"}, "/other/a.mmd", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.h.matches(tt.path); got != tt.want {
				t.Errorf("matches(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestHandlerProvisionDefaults(t *testing.T) {
	h := Handler{}
	if err := h.Provision(caddy.Context{}); err != nil {
		t.Fatalf("Provision() error = %v", err)
	}
	if h.Type != "auto" {
		t.Errorf("Type = %q, want auto", h.Type)
	}
	if h.Root != "{http.vars.root}" {
		t.Errorf("Root = %q, want {http.vars.root}", h.Root)
	}

	css := Handler{Type: "css"}
	if err := css.Provision(caddy.Context{}); err != nil {
		t.Fatalf("Provision() error = %v", err)
	}
	if css.Path != "/_diagrams/diagrams.css" {
		t.Errorf("Path = %q, want /_diagrams/diagrams.css", css.Path)
	}
}

func TestHandlerProvisionRejectsUnknownType(t *testing.T) {
	h := Handler{Type: "svg"}
	if err := h.Provision(caddy.Context{}); err == nil {
		t.Fatal("Provision() expected an error for unknown type")
	}
}

func TestHandlerServeCSS(t *testing.T) {
	h := Handler{Type: "css"}
	if err := h.Provision(caddy.Context{}); err != nil {
		t.Fatalf("Provision() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/_diagrams/diagrams.css", nil)
	if err := h.ServeHTTP(rec, req, caddyhttp.HandlerFunc(func(http.ResponseWriter, *http.Request) error {
		t.Fatal("next handler should not be called for a CSS match")
		return nil
	})); err != nil {
		t.Fatalf("ServeHTTP() error = %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/css") {
		t.Errorf("Content-Type = %q, want text/css", ct)
	}
	if !strings.Contains(rec.Body.String(), "--diagram-") {
		t.Error("CSS body does not contain the --diagram- palette")
	}
}

func TestHandlerServeDiagram(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "flow.mmd")
	if err := os.WriteFile(src, []byte("flowchart TD\n  A --> B\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := Handler{Type: "auto", Root: dir}
	if err := h.Provision(caddy.Context{}); err != nil {
		t.Fatalf("Provision() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/flow.mmd", nil)
	if err := h.ServeHTTP(rec, req, caddyhttp.HandlerFunc(func(http.ResponseWriter, *http.Request) error {
		t.Fatal("next handler should not be called for a diagram match")
		return nil
	})); err != nil {
		t.Fatalf("ServeHTTP() error = %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "svg-diagram") {
		t.Error("body does not contain a diagram figure")
	}
	if !strings.Contains(body, "<style>") {
		t.Error("body does not contain the stylesheet")
	}
}

func TestHandlerServeMarkdown(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "page.md")
	if err := os.WriteFile(src, []byte("# Title\n\n```mermaid\nflowchart TD\n  A --> B\n```\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := Handler{Type: "markdown", Root: dir}
	if err := h.Provision(caddy.Context{}); err != nil {
		t.Fatalf("Provision() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/page.md", nil)
	if err := h.ServeHTTP(rec, req, caddyhttp.HandlerFunc(func(http.ResponseWriter, *http.Request) error {
		t.Fatal("next handler should not be called for a markdown match")
		return nil
	})); err != nil {
		t.Fatalf("ServeHTTP() error = %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "svg-diagram") {
		t.Error("body does not contain a diagram figure")
	}
	if strings.Count(body, "<style>") != 1 {
		t.Errorf("expected exactly one <style> block, got %d", strings.Count(body, "<style>"))
	}
}

func TestHandlerPassesThrough(t *testing.T) {
	h := Handler{Type: "auto"}
	if err := h.Provision(caddy.Context{}); err != nil {
		t.Fatalf("Provision() error = %v", err)
	}

	called := false
	next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		called = true
		w.WriteHeader(http.StatusNoContent)
		return nil
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/index.html", nil)
	if err := h.ServeHTTP(rec, req, next); err != nil {
		t.Fatalf("ServeHTTP() error = %v", err)
	}
	if !called {
		t.Fatal("next handler was not called for a non-matching request")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
}
