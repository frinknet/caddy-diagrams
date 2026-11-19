// Package diagram integrates the goldmark-diagrams extension into Caddy.
//
// It exposes three surfaces:
//
//   - an HTTP handler directive (http.handlers.diagram) that serves diagram
//     source files as standalone SVG documents, rendered markdown, or the
//     reference stylesheet;
//   - a template extension (http.handlers.templates.functions.diagram) that
//     provides the {{ diagram }} action and the {{ diagramCSS }} helper;
//   - a markdown renderer that injects the reference stylesheet once at the
//     top of the rendered HTML when a diagram is present.
package diagram

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	diagrams "github.com/goldmark-diagrams/goldmark-diagrams"
)

const (
	// TypeAuto detects the diagram language from the file extension or the
	// source itself.
	TypeAuto = "auto"
	// TypeCSS serves the reference stylesheet instead of a diagram.
	TypeCSS = "css"
	// TypeMarkdown renders a markdown document with diagram support.
	TypeMarkdown = "markdown"
)

// Config holds the rendering options shared by the handler and the template
// extension.
type Config struct {
	// Theme is one of auto, light, dark, or neutral. Default: auto.
	Theme string `json:"theme,omitempty"`

	// ErrorMode controls how invalid diagrams are rendered: svg_card (default),
	// code_block, or fail.
	ErrorMode string `json:"error_mode,omitempty"`

	// Timeout bounds each render. Default: 5s.
	Timeout caddy.Duration `json:"timeout,omitempty"`

	// D2Layout selects the D2 layout. Only dagre is supported.
	D2Layout string `json:"d2_layout,omitempty"`

	// Languages restricts which fence languages are intercepted. Default: all.
	Languages []string `json:"languages,omitempty"`
}

// Validate checks the configuration values.
func (c *Config) Validate() error {
	switch strings.ToLower(c.Theme) {
	case "", "auto", "light", "dark", "neutral":
	default:
		return fmt.Errorf("invalid theme %q: must be auto, light, dark, or neutral", c.Theme)
	}

	switch strings.ToLower(c.ErrorMode) {
	case "", string(diagrams.ErrorSVGCard), string(diagrams.ErrorCodeBlock), string(diagrams.ErrorFail):
	default:
		return fmt.Errorf("invalid error_mode %q: must be svg_card, code_block, or fail", c.ErrorMode)
	}

	if c.Timeout < 0 {
		return fmt.Errorf("timeout must not be negative")
	}

	if c.D2Layout != "" && c.D2Layout != "dagre" {
		return fmt.Errorf("invalid d2_layout %q: only dagre is supported", c.D2Layout)
	}

	return nil
}

func (c *Config) options() []diagrams.Option {
	var opts []diagrams.Option
	if c.Theme != "" {
		opts = append(opts, diagrams.WithTheme(c.Theme))
	}
	if c.ErrorMode != "" {
		opts = append(opts, diagrams.WithErrorMode(diagrams.ErrorMode(c.ErrorMode)))
	}
	if c.Timeout > 0 {
		opts = append(opts, diagrams.WithTimeout(time.Duration(c.Timeout)))
	}
	if c.D2Layout != "" {
		opts = append(opts, diagrams.WithD2Layout(c.D2Layout))
	}
	if len(c.Languages) > 0 {
		opts = append(opts, diagrams.WithLanguages(c.Languages...))
	}
	return opts
}

func (c *Config) timeout() time.Duration {
	if c.Timeout > 0 {
		return time.Duration(c.Timeout)
	}
	return 5 * time.Second
}

// NormalizeType lowercases and trims a diagram type.
func NormalizeType(t string) string {
	return strings.ToLower(strings.TrimSpace(t))
}

// SupportedType reports whether t is a diagram language, auto, css, or markdown.
func SupportedType(t string) bool {
	switch NormalizeType(t) {
	case TypeAuto, TypeCSS, TypeMarkdown,
		"mermaid", "mmd", "d2", "plantuml", "puml",
		"dot", "graphviz", "chart", "echarts":
		return true
	}
	return false
}

// IsDiagramLanguage reports whether t is a supported diagram language.
func IsDiagramLanguage(t string) bool {
	switch NormalizeType(t) {
	case "mermaid", "mmd", "d2", "plantuml", "puml",
		"dot", "graphviz", "chart", "echarts":
		return true
	}
	return false
}

// IsDiagramPath reports whether p has a diagram source extension.
func IsDiagramPath(p string) bool {
	switch strings.ToLower(path.Ext(p)) {
	case ".mmd", ".mermaid", ".d2", ".puml", ".plantuml",
		".dot", ".gv", ".graphviz", ".chart", ".echarts":
		return true
	}
	return false
}

// DetectLanguage picks a diagram language from the file path and source. The
// path extension wins; otherwise the source is inspected with simple prefixes.
func DetectLanguage(filePath, source string) string {
	switch strings.ToLower(path.Ext(filePath)) {
	case ".mmd", ".mermaid":
		return "mermaid"
	case ".d2":
		return "d2"
	case ".puml", ".plantuml":
		return "plantuml"
	case ".dot", ".gv", ".graphviz":
		return "dot"
	case ".chart", ".echarts":
		return "chart"
	}

	trimmed := strings.TrimSpace(source)
	lower := strings.ToLower(trimmed)

	switch {
	case strings.HasPrefix(lower, "@startuml"):
		return "plantuml"
	case strings.HasPrefix(lower, "digraph"), strings.HasPrefix(lower, "graph"), strings.HasPrefix(lower, "strict"):
		return "dot"
	case strings.HasPrefix(lower, "{"), strings.HasPrefix(lower, "["):
		return "chart"
	}

	for _, prefix := range []string{
		"flowchart", "graph ", "sequencediagram", "classdiagram",
		"statediagram", "erdiagram", "journey", "gantt", "pie",
		"mindmap", "timeline", "gitgraph", "quadrantchart",
		"xychart", "kanban", "sankey", "block", "packet",
		"architecture", "radar", "treemap", "requirement", "c4context",
	} {
		if strings.HasPrefix(lower, prefix) {
			return "mermaid"
		}
	}

	return "mermaid"
}

// RenderStandalone renders one diagram as a standalone HTML fragment.
func RenderStandalone(ctx context.Context, lang, source string, opts ...diagrams.Option) (string, error) {
	return diagrams.RenderStandalone(ctx, lang, source, opts...)
}

// DefaultCSS returns the reference stylesheet.
func DefaultCSS() string {
	return diagrams.DefaultCSS()
}
