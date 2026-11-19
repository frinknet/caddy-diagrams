package diagram

import (
	"context"
	"strings"
	"testing"
)

func TestRenderMarkdownInjectsCSSOnce(t *testing.T) {
	src := "# Title\n\n```mermaid\nflowchart TD\n  A --> B\n```\n"
	out, err := RenderMarkdown(context.Background(), src)
	if err != nil {
		t.Fatalf("RenderMarkdown() error = %v", err)
	}
	if !strings.Contains(out, "svg-diagram") {
		t.Fatalf("rendered output does not contain a diagram: %s", out)
	}
	if strings.Count(out, "<style>") != 1 {
		t.Fatalf("expected exactly one <style> block, got %d", strings.Count(out, "<style>"))
	}
	if !strings.HasPrefix(out, "<style>") {
		t.Fatalf("expected the stylesheet at the beginning, got: %.40s", out)
	}
}

func TestRenderMarkdownWithoutDiagramHasNoCSS(t *testing.T) {
	out, err := RenderMarkdown(context.Background(), "# Title\n\nplain paragraph\n")
	if err != nil {
		t.Fatalf("RenderMarkdown() error = %v", err)
	}
	if strings.Contains(out, "<style>") {
		t.Fatalf("expected no stylesheet without a diagram, got: %s", out)
	}
}

func TestHasDiagram(t *testing.T) {
	if !HasDiagram(`<figure class="svg-diagram">`) {
		t.Error("HasDiagram() = false, want true")
	}
	if HasDiagram(`<p>plain</p>`) {
		t.Error("HasDiagram() = true, want false")
	}
}
