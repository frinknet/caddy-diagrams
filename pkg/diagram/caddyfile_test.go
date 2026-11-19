package diagram

import (
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

func TestHandlerUnmarshalCaddyfile(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   string
		want Handler
	}{
		{
			name: "type only",
			in:   "diagram auto",
			want: Handler{Type: "auto"},
		},
		{
			name: "type and path",
			in:   "diagram mermaid /docs/",
			want: Handler{Type: "mermaid", Path: "/docs/"},
		},
		{
			name: "block options",
			in: `diagram dot {
				theme dark
				error_mode fail
				timeout 2s
				d2_layout dagre
			}`,
			want: Handler{
				Type: "dot",
				Config: Config{
					Theme:     "dark",
					ErrorMode: "fail",
					Timeout:   caddy.Duration(2 * time.Second),
					D2Layout:  "dagre",
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var h Handler
			if err := h.UnmarshalCaddyfile(caddyfile.NewTestDispenser(tt.in)); err != nil {
				t.Fatalf("UnmarshalCaddyfile() error = %v", err)
			}
			if h.Type != tt.want.Type {
				t.Errorf("Type = %q, want %q", h.Type, tt.want.Type)
			}
			if h.Path != tt.want.Path {
				t.Errorf("Path = %q, want %q", h.Path, tt.want.Path)
			}
			if h.Theme != tt.want.Theme {
				t.Errorf("Theme = %q, want %q", h.Theme, tt.want.Theme)
			}
			if h.ErrorMode != tt.want.ErrorMode {
				t.Errorf("ErrorMode = %q, want %q", h.ErrorMode, tt.want.ErrorMode)
			}
			if h.Timeout != tt.want.Timeout {
				t.Errorf("Timeout = %v, want %v", h.Timeout, tt.want.Timeout)
			}
			if h.D2Layout != tt.want.D2Layout {
				t.Errorf("D2Layout = %q, want %q", h.D2Layout, tt.want.D2Layout)
			}
		})
	}
}

func TestHandlerUnmarshalCaddyfileRejectsExtraArgs(t *testing.T) {
	var h Handler
	if err := h.UnmarshalCaddyfile(caddyfile.NewTestDispenser("diagram auto one two")); err == nil {
		t.Fatal("UnmarshalCaddyfile() expected an error for extra arguments")
	}
}

func TestTemplateFunctionsUnmarshalCaddyfile(t *testing.T) {
	var tf TemplateFunctions
	if err := tf.UnmarshalCaddyfile(caddyfile.NewTestDispenser(`diagram {
		theme dark
		error_mode code_block
		timeout 3s
	}`)); err != nil {
		t.Fatalf("UnmarshalCaddyfile() error = %v", err)
	}
	if tf.Theme != "dark" {
		t.Errorf("Theme = %q, want dark", tf.Theme)
	}
	if tf.ErrorMode != "code_block" {
		t.Errorf("ErrorMode = %q, want code_block", tf.ErrorMode)
	}
	if tf.Timeout != caddy.Duration(3*time.Second) {
		t.Errorf("Timeout = %v, want 3s", tf.Timeout)
	}
}
