package diagram

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	httpcaddyfile.RegisterHandlerDirective("diagram", parseCaddyfile)
	httpcaddyfile.RegisterDirectiveOrder("diagram", httpcaddyfile.Before, "file_server")
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var handler Handler
	if err := handler.UnmarshalCaddyfile(h.Dispenser); err != nil {
		return nil, err
	}
	return &handler, nil
}

// UnmarshalCaddyfile parses the diagram directive. Syntax:
//
//	diagram [<matcher>] <type> [<path>] {
//		root <path>
//		theme <auto|light|dark|neutral>
//		error_mode <svg_card|code_block|fail>
//		timeout <duration>
//		d2_layout <dagre>
//		languages <lang...>
//	}
func (h *Handler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next() // consume directive name

	if !d.Next() {
		return d.ArgErr()
	}
	h.Type = d.Val()

	if d.NextArg() {
		h.Path = d.Val()
	}
	if d.NextArg() {
		return d.ArgErr()
	}

	for nesting := d.Nesting(); d.NextBlock(nesting); {
		switch d.Val() {
		case "root":
			if !d.NextArg() {
				return d.ArgErr()
			}
			h.Root = d.Val()
		case "theme":
			if !d.NextArg() {
				return d.ArgErr()
			}
			h.Theme = d.Val()
		case "error_mode":
			if !d.NextArg() {
				return d.ArgErr()
			}
			h.ErrorMode = d.Val()
		case "timeout":
			if !d.NextArg() {
				return d.ArgErr()
			}
			dur, err := caddy.ParseDuration(d.Val())
			if err != nil {
				return d.Errf("invalid timeout: %v", err)
			}
			h.Timeout = caddy.Duration(dur)
		case "d2_layout":
			if !d.NextArg() {
				return d.ArgErr()
			}
			h.D2Layout = d.Val()
		case "languages":
			h.Languages = d.RemainingArgs()
		default:
			return d.Errf("unrecognized option %q", d.Val())
		}
	}
	return nil
}

// UnmarshalCaddyfile parses the template extension. Syntax:
//
//	diagram {
//		theme <auto|light|dark|neutral>
//		error_mode <svg_card|code_block|fail>
//		timeout <duration>
//		d2_layout <dagre>
//		languages <lang...>
//	}
func (t *TemplateFunctions) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next() // consume module name

	for nesting := d.Nesting(); d.NextBlock(nesting); {
		switch d.Val() {
		case "theme":
			if !d.NextArg() {
				return d.ArgErr()
			}
			t.Theme = d.Val()
		case "error_mode":
			if !d.NextArg() {
				return d.ArgErr()
			}
			t.ErrorMode = d.Val()
		case "timeout":
			if !d.NextArg() {
				return d.ArgErr()
			}
			dur, err := caddy.ParseDuration(d.Val())
			if err != nil {
				return d.Errf("invalid timeout: %v", err)
			}
			t.Timeout = caddy.Duration(dur)
		case "d2_layout":
			if !d.NextArg() {
				return d.ArgErr()
			}
			t.D2Layout = d.Val()
		case "languages":
			t.Languages = d.RemainingArgs()
		default:
			return d.Errf("unrecognized option %q", d.Val())
		}
	}
	return nil
}
