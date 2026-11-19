package diagram

import (
	"context"
	"fmt"
	"html/template"

	"github.com/caddyserver/caddy/v2"
	diagrams "github.com/frinknet/caddy-diagrams/pkg/goldmark-diagrams"
)

func init() {
	caddy.RegisterModule(TemplateFunctions{})
}

// TemplateFunctions adds the {{ diagram }} action and {{ diagramCSS }} helper to
// the templates handler. Register it from the Caddyfile with:
//
//	templates {
//		extensions {
//			diagram
//		}
//	}
type TemplateFunctions struct {
	Config
}

// CaddyModule returns the Caddy module information.
func (TemplateFunctions) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.templates.functions.diagram",
		New: func() caddy.Module { return new(TemplateFunctions) },
	}
}

// Provision validates the template extension configuration.
func (t *TemplateFunctions) Provision(_ caddy.Context) error {
	return t.Config.Validate()
}

// Validate validates the template extension configuration.
func (t *TemplateFunctions) Validate() error {
	return t.Config.Validate()
}

// CustomTemplateFunctions implements the templates.CustomFunctions interface.
func (t *TemplateFunctions) CustomTemplateFunctions() template.FuncMap {
	return template.FuncMap{
		"diagram":         t.funcDiagram,
		"diagramCSS":      t.funcDiagramCSS,
		"diagramMarkdown": t.funcDiagramMarkdown,
	}
}

// funcDiagram renders {{ diagram type body }}. The css type returns the
// reference stylesheet wrapped in a style element; markdown renders the body as
// a markdown document; every diagram language renders a standalone figure.
func (t *TemplateFunctions) funcDiagram(typ, body string) (template.HTML, error) {
	typ = NormalizeType(typ)

	switch typ {
	case TypeCSS:
		return template.HTML("<style>" + diagrams.DefaultCSS() + "</style>"), nil
	case TypeMarkdown:
		return t.funcDiagramMarkdown(body)
	case TypeAuto:
		typ = DetectLanguage("", body)
	}

	if !IsDiagramLanguage(typ) {
		return "", fmt.Errorf("unsupported diagram type %q", typ)
	}

	ctx, cancel := context.WithTimeout(context.Background(), t.timeout())
	defer cancel()

	fragment, err := diagrams.RenderStandalone(ctx, typ, body, t.Config.options()...)
	if err != nil {
		return "", err
	}
	return template.HTML(fragment), nil
}

// funcDiagramCSS returns the reference stylesheet wrapped in a style element.
func (t *TemplateFunctions) funcDiagramCSS() template.HTML {
	return template.HTML("<style>" + diagrams.DefaultCSS() + "</style>")
}

// funcDiagramMarkdown renders a markdown document with diagram support and
// injects the stylesheet once when a diagram is present.
func (t *TemplateFunctions) funcDiagramMarkdown(body string) (template.HTML, error) {
	ctx, cancel := context.WithTimeout(context.Background(), t.timeout())
	defer cancel()

	out, err := RenderMarkdown(ctx, body, t.Config.options()...)
	if err != nil {
		return "", err
	}
	return template.HTML(out), nil
}
