package diagram

import (
	"testing"
)

func TestSupportedType(t *testing.T) {
	for _, tt := range []string{"auto", "css", "markdown", "mermaid", "mmd", "d2", "plantuml", "puml", "dot", "graphviz", "chart", "echarts", "AUTO", "Mermaid"} {
		if !SupportedType(tt) {
			t.Errorf("SupportedType(%q) = false, want true", tt)
		}
	}
	for _, tt := range []string{"", "svg", "json", "nope"} {
		if SupportedType(tt) {
			t.Errorf("SupportedType(%q) = true, want false", tt)
		}
	}
}

func TestIsDiagramLanguage(t *testing.T) {
	for _, tt := range []string{"mermaid", "mmd", "d2", "plantuml", "puml", "dot", "graphviz", "chart", "echarts"} {
		if !IsDiagramLanguage(tt) {
			t.Errorf("IsDiagramLanguage(%q) = false, want true", tt)
		}
	}
	for _, tt := range []string{"auto", "css", "markdown", ""} {
		if IsDiagramLanguage(tt) {
			t.Errorf("IsDiagramLanguage(%q) = true, want false", tt)
		}
	}
}

func TestIsDiagramPath(t *testing.T) {
	for _, tt := range []string{"/a.mmd", "/a.mermaid", "/a.d2", "/a.puml", "/a.plantuml", "/a.dot", "/a.gv", "/a.graphviz", "/a.chart", "/a.echarts", "/A.MMD"} {
		if !IsDiagramPath(tt) {
			t.Errorf("IsDiagramPath(%q) = false, want true", tt)
		}
	}
	for _, tt := range []string{"/a.md", "/a.html", "/a", "/a.svg"} {
		if IsDiagramPath(tt) {
			t.Errorf("IsDiagramPath(%q) = true, want false", tt)
		}
	}
}

func TestDetectLanguage(t *testing.T) {
	for _, tt := range []struct {
		path string
		src  string
		want string
	}{
		{"/a.mmd", "", "mermaid"},
		{"/a.mermaid", "", "mermaid"},
		{"/a.d2", "", "d2"},
		{"/a.puml", "", "plantuml"},
		{"/a.dot", "", "dot"},
		{"/a.gv", "", "dot"},
		{"/a.chart", "", "chart"},
		{"", "digraph G { A -> B }", "dot"},
		{"", "graph G { A -- B }", "dot"},
		{"", "strict digraph G { A -> B }", "dot"},
		{"", "@startuml\nA -> B\n@enduml", "plantuml"},
		{"", "flowchart TD\n A --> B", "mermaid"},
		{"", "sequenceDiagram\n A->>B: hi", "mermaid"},
		{"", "classDiagram\n A <|-- B", "mermaid"},
		{"", "gantt\n title A", "mermaid"},
		{"", "kanban\n Todo\n  [Task]", "mermaid"},
		{"", "not a diagram", "mermaid"},
	} {
		if got := DetectLanguage(tt.path, tt.src); got != tt.want {
			t.Errorf("DetectLanguage(%q, %q) = %q, want %q", tt.path, tt.src, got, tt.want)
		}
	}
}

func TestConfigValidate(t *testing.T) {
	for _, tt := range []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"zero", Config{}, false},
		{"theme auto", Config{Theme: "auto"}, false},
		{"theme dark", Config{Theme: "dark"}, false},
		{"theme bad", Config{Theme: "neon"}, true},
		{"error mode svg_card", Config{ErrorMode: "svg_card"}, false},
		{"error mode code_block", Config{ErrorMode: "code_block"}, false},
		{"error mode fail", Config{ErrorMode: "fail"}, false},
		{"error mode bad", Config{ErrorMode: "nope"}, true},
		{"negative timeout", Config{Timeout: -1}, true},
		{"d2 dagre", Config{D2Layout: "dagre"}, false},
		{"d2 elk", Config{D2Layout: "elk"}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
