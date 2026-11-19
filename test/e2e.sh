#!/bin/sh
# e2e test runner — boots the image ./build.sh produces and exercises the
# diagram handler, the template extension, markdown rendering, and the CSS
# endpoint. It never builds the image.
#
#   sh test/e2e.sh
#
# E2E_IMAGE overrides the image (default: the NOTICE slug with a :test tag).
# E2E_UPDATE=1 rewrites test/e2e/fixtures/expected/ from the responses.
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT"

TMP=$(mktemp -d /tmp/caddy-diagrams-e2e-XXXXXX)
PORT=9099
SLUG=$(sed -n 's/^Caddy Plugin: //p' ./NOTICE)
IMAGE="${E2E_IMAGE:-${SLUG}:test}"
CONTAINER_NAME="caddydiagrams-e2e-runner"
PASS=0
FAIL=0

cleanup() {
	timeout 20 docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
	rm -rf "$TMP" 2>/dev/null || true
}
trap cleanup EXIT INT

pass() {
	PASS=$((PASS + 1))
	printf '  PASS: %s\n' "$1"
}

fail() {
	FAIL=$((FAIL + 1))
	printf '  FAIL: %s\n' "$1"
	if [ "$#" -gt 1 ]; then
		printf '    %s\n' "$2"
	fi
}

check_eq() {
	if [ "$2" = "$3" ]; then
		pass "$1"
	else
		fail "$1" "expected: $2, got: $3"
	fi
}

check_has() {
	if printf '%s' "$3" | grep -qF -- "$2"; then
		pass "$1"
	else
		fail "$1" "missing: $2"
	fi
}

check_absent() {
	if printf '%s' "$3" | grep -qF -- "$2"; then
		fail "$1" "unexpected: $2"
	else
		pass "$1"
	fi
}

check_file() {
	EXPECTED="test/e2e/fixtures/expected/$2"
	if [ "${E2E_UPDATE:-0}" = "1" ]; then
		mkdir -p "$(dirname "$EXPECTED")"
		cp "$TMP/body" "$EXPECTED"
	fi
	if cmp -s "$TMP/body" "$EXPECTED"; then
		pass "$1"
	else
		fail "$1" "body differs from expected/$2"
	fi
}

request() {
	REQ_PORT=$1
	REQ_PATH=$2
	STATUS=$(timeout 15 curl -sS --max-time 10 -o "$TMP/body" -w '%{http_code}' "http://127.0.0.1:$REQ_PORT$REQ_PATH" 2>/dev/null) || STATUS=000
}

body_has() {
	check_has "$1" "$2" "$(cat "$TMP/body" 2>/dev/null || true)"
}

body_absent() {
	check_absent "$1" "$2" "$(cat "$TMP/body" 2>/dev/null || true)"
}

wait_ready() {
	WAIT_PORT=$1
	WAIT_I=0
	while [ "$WAIT_I" -lt 40 ]; do
		if timeout 5 curl -sf --max-time 2 "http://127.0.0.1:$WAIT_PORT/index.html" >/dev/null 2>&1; then
			return 0
		fi
		WAIT_I=$((WAIT_I + 1))
		sleep 0.25
	done
	return 1
}

start_container() {
	START_CONFIG=$1
	START_WWW=$2
	START_PORT=$3
	timeout 60 docker run -d --no-healthcheck --name "$CONTAINER_NAME" \
		-v "$START_CONFIG:/etc/caddy/Caddyfile:ro" \
		-v "$START_WWW:/srv/www:ro" \
		-p "127.0.0.1:$START_PORT:80" \
		"$IMAGE" run --config /etc/caddy/Caddyfile --adapter caddyfile >/dev/null
}

validate_ok() {
	VALIDATE_DESC=$1
	VALIDATE_CFG=$2
	VALIDATE_STATUS=0
	VALIDATE_OUT=$(timeout 60 docker run --rm --entrypoint /bin/caddy \
		-v "$VALIDATE_CFG:/etc/caddy/Caddyfile:ro" \
		"$IMAGE" validate --config /etc/caddy/Caddyfile --adapter caddyfile 2>&1) || VALIDATE_STATUS=$?
	if [ "$VALIDATE_STATUS" -eq 0 ]; then
		pass "$VALIDATE_DESC"
	else
		fail "$VALIDATE_DESC" "$VALIDATE_OUT"
	fi
}

validate_fail() {
	VALIDATE_DESC=$1
	VALIDATE_CFG=$2
	VALIDATE_NEEDLE=$3
	VALIDATE_STATUS=0
	VALIDATE_OUT=$(timeout 60 docker run --rm --entrypoint /bin/caddy \
		-v "$VALIDATE_CFG:/etc/caddy/Caddyfile:ro" \
		"$IMAGE" validate --config /etc/caddy/Caddyfile --adapter caddyfile 2>&1) || VALIDATE_STATUS=$?
	if [ "$VALIDATE_STATUS" -ne 0 ] && printf '%s' "$VALIDATE_OUT" | grep -qF -- "$VALIDATE_NEEDLE"; then
		pass "$VALIDATE_DESC"
	else
		fail "$VALIDATE_DESC" "exit=$VALIDATE_STATUS output=$VALIDATE_OUT"
	fi
}

timeout 20 docker image inspect "$IMAGE" >/dev/null 2>&1 || {
	echo "test/e2e.sh: image $IMAGE not found; run ./build.sh first" >&2
	exit 1
}

# ── Fixtures ──────────────────────────────────────────────────────────────
cp test/e2e/fixtures/Caddyfile "$TMP/Caddyfile"
cp -R test/e2e/fixtures/www "$TMP/www"

# ── Test 0: image contents ────────────────────────────────────────────────
echo ""
echo "=== Test 0: image contents ==="

if ! VERSIONS=$(timeout 60 docker run --rm --entrypoint /bin/cat "$IMAGE" /usr/share/licenses/versions.txt 2>&1); then
	fail "read image versions" "$VERSIONS"
else
	WANT=$(sed -n 's/^require github.com\/caddyserver\/caddy\/v2 \(v[0-9][^ ]*\).*/\1/p' go.mod | head -1)
	GOT=$(printf '%s' "$VERSIONS" | sed -n 's/^github.com\/caddyserver\/caddy\/v2 \(v[0-9][^ ]*\).*/\1/p' | head -1)
	check_eq "caddy version matches go.mod" "$WANT" "$GOT"
fi

if ! MODULES=$(timeout 60 docker run --rm --entrypoint /bin/caddy "$IMAGE" list-modules 2>&1); then
	fail "list modules" "$MODULES"
else
	for module in http.handlers.diagram http.handlers.templates.functions.diagram; do
		if printf '%s' "$MODULES" | grep -qx "$module"; then
			pass "module $module is registered"
		else
			fail "module $module is registered"
		fi
	done
fi

# ── Start Caddy ───────────────────────────────────────────────────────────
echo ""
echo "=== Starting Caddy container ==="
start_container "$TMP/Caddyfile" "$TMP/www" "$PORT"
if ! wait_ready "$PORT"; then
	echo "test/e2e.sh: container did not become ready" >&2
	timeout 20 docker logs "$CONTAINER_NAME" 2>&1 | tail -n 40 >&2 || true
	exit 1
fi
echo "Caddy ready"

# ── Test 1: static file passthrough ───────────────────────────────────────
echo ""
echo "=== Test 1: static file passthrough ==="
request "$PORT" /index.html
check_eq "static file status" "200" "$STATUS"
body_has "static file body" "static page"
body_absent "static file has no diagram" "svg-diagram"
check_file "static file exact bytes" "index.html"

# ── Test 2: standalone diagram (auto) ─────────────────────────────────────
echo ""
echo "=== Test 2: standalone diagram ==="
request "$PORT" /flow.mmd
check_eq "mermaid status" "200" "$STATUS"
body_has "mermaid has a figure" "svg-diagram"
body_has "mermaid has an svg" "<svg"
body_has "mermaid has the stylesheet" "--diagram-"
body_has "mermaid has a doctype" "<!DOCTYPE html>"
check_file "mermaid exact bytes" "flow.mmd"

request "$PORT" /flow.d2
check_eq "d2 status" "200" "$STATUS"
body_has "d2 has a figure" "svg-diagram"
body_has "d2 has an svg" "<svg"
body_has "d2 has the stylesheet" "--diagram-"
check_file "d2 exact bytes" "flow.d2"

request "$PORT" /fail.mmd
check_eq "error_mode fail status" "500" "$STATUS"
body_absent "error_mode fail has no figure" "svg-diagram"

# ── Test 3: markdown ──────────────────────────────────────────────────────
echo ""
echo "=== Test 3: markdown ==="
request "$PORT" /page.md
check_eq "markdown status" "200" "$STATUS"
body_has "markdown has a heading" "<h1"
body_has "markdown has a figure" "svg-diagram"
body_has "markdown has an svg" "<svg"
body_has "markdown has the stylesheet" "--diagram-"
body_has "markdown stylesheet is first" "<style>"
check_file "markdown exact bytes" "page.md"

request "$PORT" /plain.md
check_eq "plain markdown status" "200" "$STATUS"
body_has "plain markdown has a heading" "<h1"
body_absent "plain markdown has no stylesheet" "--diagram-"
body_absent "plain markdown has no style block" "<style>"
check_file "plain markdown exact bytes" "plain.md"

# ── Test 4: template extension ────────────────────────────────────────────
echo ""
echo "=== Test 4: template extension ==="
request "$PORT" /template.html
check_eq "template status" "200" "$STATUS"
body_has "template has a figure" "svg-diagram"
body_has "template has an svg" "<svg"
body_has "template has the stylesheet" "--diagram-"
check_file "template exact bytes" "template.html"

# ── Test 5: CSS endpoint ──────────────────────────────────────────────────
echo ""
echo "=== Test 5: CSS endpoint ==="
request "$PORT" /_diagrams/diagrams.css
check_eq "css status" "200" "$STATUS"
body_has "css has the palette" "--diagram-"
body_absent "css is not an html document" "<!DOCTYPE html>"
check_file "css exact bytes" "diagrams.css"

# ── Test 6: missing file ──────────────────────────────────────────────────
echo ""
echo "=== Test 6: missing file ==="
request "$PORT" /missing.html
check_eq "missing status" "404" "$STATUS"

# ── Test 7: validation ────────────────────────────────────────────────────
echo ""
echo "=== Test 7: validation ==="

cat > "$TMP/valid.Caddyfile" << 'CADDY'
{
	admin off
}

:80 {
	root * /srv/www
	handle /_diagrams/diagrams.css {
		diagram css
	}
	handle /flow.mmd {
		diagram auto
	}
	handle {
		file_server
	}
}
CADDY
validate_ok "valid Caddyfile" "$TMP/valid.Caddyfile"

cat > "$TMP/bad-type.Caddyfile" << 'CADDY'
{
	admin off
}

:80 {
	diagram nope
}
CADDY
validate_fail "unknown type fails" "$TMP/bad-type.Caddyfile" "unsupported diagram type"

cat > "$TMP/bad-theme.Caddyfile" << 'CADDY'
{
	admin off
}

:80 {
	diagram auto {
		theme neon
	}
}
CADDY
validate_fail "unknown theme fails" "$TMP/bad-theme.Caddyfile" "invalid theme"

cat > "$TMP/bad-option.Caddyfile" << 'CADDY'
{
	admin off
}

:80 {
	diagram auto {
		nonsense on
	}
}
CADDY
validate_fail "unknown option fails" "$TMP/bad-option.Caddyfile" "unrecognized option"

# ── Summary ───────────────────────────────────────────────────────────────
echo ""
echo "=============================================="
printf '  PASS: %d  FAIL: %d\n' "$PASS" "$FAIL"
echo "=============================================="
if [ "$FAIL" -ne 0 ]; then
	echo ""
	echo "=== container logs ==="
	timeout 20 docker logs "$CONTAINER_NAME" 2>&1 | tail -n 40 || true
	exit 1
fi
