# caddy-diagrams demo

This page is rendered by the `diagram markdown` handler. The stylesheet is
injected once at the top of the page.

```mermaid
flowchart TD
  A[Request] --> B{Markdown?}
  B -- yes --> C[Render with diagrams]
  B -- no --> D[file_server]
  C --> E[HTML with SVG]
```

The standalone diagram is served at `/flow.mmd` and the stylesheet at
`/_diagrams/diagrams.css`.
