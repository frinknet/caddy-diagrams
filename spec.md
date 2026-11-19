simply add https://github.com/frinknet/goldmark-diagrams to caddy goldmark.

I'd also like to add a template directive {{ diagram type body }}

and also a caddyfile directive

@route {
  diagram type
  file_server
}

type can be auto and then it autodetects from the data you send.

we also have a type css that provides the css for the svg. 

we need to think how that works when it's sered as an independent file.  Probably that is differentthan when served in a template. and when served though markdown we need to send the CSS block once at the beginning of the md or else not at all. 

is that clear

