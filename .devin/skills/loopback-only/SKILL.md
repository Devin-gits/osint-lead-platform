# Loopback-only servers — no network exposure

## Purpose

All local development servers (UI, API, any tooling) MUST bind to `127.0.0.1` only.
Never expose ports on `0.0.0.0` or LAN interfaces. This is a pre-production OSINT platform
with real lead-shaped data — no port should be reachable from outside the local machine.

## Rules

### Next.js UI

Always pass `-H 127.0.0.1`:

```bash
# Development
npx next dev -H 127.0.0.1 -p 3000

# Production-style local
npx next start -H 127.0.0.1 -p 3000
```

Never use `next dev` or `next start` without `-H 127.0.0.1`.
Ignore the "Network: http://10.x.x.x:..." line — it must not be relied upon or advertised.

### Control-plane API (Go)

Start with loopback binding:

```bash
LISTEN_ADDR=127.0.0.1:8080 go run ./cmd/server
```

If the Go server does not yet support `LISTEN_ADDR`, use:

```bash
go run ./cmd/server
# Accept that it binds 0.0.0.0:8080 for now, but:
# - Do NOT advertise the LAN URL
# - Do NOT rely on external access
# - Document that LISTEN_ADDR support is needed
```

### Playwright / E2E tests

All E2E tests connect to `http://localhost:3000` and `http://localhost:8080`.
These resolve to `127.0.0.1` — no LAN access needed.

### General principles

1. **No network bind required** — Devin works entirely locally via loopback
2. **No port forwarding** — never set up tunnels, ngrok, or expose services externally
3. **curl / fetch always use localhost** — never use LAN IPs in scripts or tests
4. **CORS stays as `http://localhost:3000`** — do not widen to `*` or LAN origins
5. **When starting any server process**, always prefer explicit `127.0.0.1` bind
6. **If a tool prints a LAN URL**, ignore it — it is informational only, not a requirement

## Startup template for E2E verification

```bash
# Kill any existing servers
pkill -f 'next start' 2>/dev/null
pkill -f 'next dev' 2>/dev/null
pkill -f 'go run ./cmd/server' 2>/dev/null
sleep 1

# Start API (loopback)
cd services/control-plane
LISTEN_ADDR=127.0.0.1:8080 nohup go run ./cmd/server > /tmp/api.log 2>&1 &

# Start UI (loopback)
cd ui/web-console
nohup npx next start -H 127.0.0.1 -p 3000 > /tmp/ui.log 2>&1 &

# Verify
sleep 5
curl -s -o /dev/null -w '%{http_code}' http://localhost:3000 && echo " ui ok"
curl -s -o /dev/null -w '%{http_code}' http://localhost:8080/api/leads && echo " api ok"
```

## Future improvement (not blocking)

The Go control-plane should support `LISTEN_ADDR` env var to bind `127.0.0.1:{port}`
by default, with override for real deployment. Until then, document the limitation.
