package diagram

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	diagrams "github.com/frinknet/goldmark-diagrams"
)

func init() {
	caddy.RegisterModule(Handler{})
}

// Handler serves diagram source files, rendered markdown, or the reference
// stylesheet. It is intended to sit in a route ahead of file_server: requests
// it recognizes are answered here, everything else passes through.
type Handler struct {
	// Type is the diagram language, auto, css, or markdown. Default: auto.
	Type string `json:"type,omitempty"`

	// Path restricts the handler to an exact path, a prefix ending in "/", or a
	// glob. When empty, the type decides: css serves /_diagrams/diagrams.css,
	// markdown serves *.md, and diagram languages serve known source extensions.
	Path string `json:"path,omitempty"`

	// Root is the directory diagram sources are read from. Default:
	// {http.vars.root}.
	Root string `json:"root,omitempty"`

	Config
}

// CaddyModule returns the Caddy module information.
func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.diagram",
		New: func() caddy.Module { return new(Handler) },
	}
}

// Provision sets defaults and validates the handler.
func (h *Handler) Provision(_ caddy.Context) error {
	if h.Type == "" {
		h.Type = TypeAuto
	}
	h.Type = NormalizeType(h.Type)
	if !SupportedType(h.Type) {
		return fmt.Errorf("unsupported diagram type %q", h.Type)
	}
	if h.Type == TypeCSS && h.Path == "" {
		h.Path = "/_diagrams/diagrams.css"
	}
	if h.Root == "" {
		h.Root = "{http.vars.root}"
	}
	return h.Config.Validate()
}

// Validate validates the handler configuration.
func (h *Handler) Validate() error {
	if !SupportedType(h.Type) {
		return fmt.Errorf("unsupported diagram type %q", h.Type)
	}
	return h.Config.Validate()
}

// ServeHTTP serves the matching request or passes it to the next handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	if !h.matches(r.URL.Path) {
		return next.ServeHTTP(w, r)
	}

	switch h.Type {
	case TypeCSS:
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_, err := w.Write([]byte(diagrams.DefaultCSS()))
		return err
	case TypeMarkdown:
		return h.serveMarkdown(w, r, next)
	default:
		return h.serveDiagram(w, r, next)
	}
}

func (h *Handler) matches(reqPath string) bool {
	if h.Path != "" {
		if strings.HasSuffix(h.Path, "/") {
			return strings.HasPrefix(reqPath, h.Path)
		}
		if strings.ContainsAny(h.Path, "*?[") {
			ok, _ := path.Match(h.Path, reqPath)
			return ok
		}
		return reqPath == h.Path
	}

	switch h.Type {
	case TypeCSS:
		return reqPath == "/_diagrams/diagrams.css"
	case TypeMarkdown:
		return strings.HasSuffix(strings.ToLower(reqPath), ".md")
	default:
		return IsDiagramPath(reqPath)
	}
}

func (h *Handler) serveDiagram(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	filename := h.filename(r)
	data, err := os.ReadFile(filename)
	if err != nil {
		return next.ServeHTTP(w, r)
	}

	lang := h.Type
	if lang == TypeAuto {
		lang = DetectLanguage(r.URL.Path, string(data))
	}
	if !IsDiagramLanguage(lang) {
		return next.ServeHTTP(w, r)
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout())
	defer cancel()

	fragment, err := diagrams.RenderStandalone(ctx, lang, string(data), h.options()...)
	if err != nil {
		return caddyhttp.Error(http.StatusInternalServerError, err)
	}

	doc := "<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><style>" +
		diagrams.DefaultCSS() + "</style></head><body>" + fragment + "</body></html>\n"

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, err = w.Write([]byte(doc))
	return err
}

func (h *Handler) serveMarkdown(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	filename := h.filename(r)
	data, err := os.ReadFile(filename)
	if err != nil {
		return next.ServeHTTP(w, r)
	}

	out, err := RenderMarkdown(r.Context(), string(data), h.options()...)
	if err != nil {
		return caddyhttp.Error(http.StatusInternalServerError, err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, err = w.Write([]byte(out))
	return err
}

func (h *Handler) filename(r *http.Request) string {
	root := h.Root
	if repl, ok := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer); ok {
		root = repl.ReplaceAll(root, "")
	}
	return caddyhttp.SanitizedPathJoin(root, r.URL.Path)
}

func (h *Handler) timeout() time.Duration {
	if h.Timeout > 0 {
		return time.Duration(h.Timeout)
	}
	return 5 * time.Second
}
