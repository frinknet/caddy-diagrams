package diagram

import (
	"bytes"
	"context"
	"strings"

	diagrams "github.com/frinknet/caddy-diagrams/pkg/goldmark-diagrams"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

// RenderMarkdown renders a markdown document with diagram support. When the
// output contains a diagram, the reference stylesheet is prepended exactly once
// so the rendered page styles itself without a second request.
func RenderMarkdown(ctx context.Context, source string, opts ...diagrams.Option) (string, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			diagrams.New(opts...),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			gmhtml.WithUnsafe(),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		return "", err
	}

	out := buf.String()
	if strings.Contains(out, "svg-diagram") {
		out = "<style>" + diagrams.DefaultCSS() + "</style>\n" + out
	}
	return out, nil
}

// HasDiagram reports whether rendered HTML contains a diagram figure.
func HasDiagram(html string) bool {
	return strings.Contains(html, "svg-diagram")
}
