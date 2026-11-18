# BrainLogistics tracking server

Go standard library HTTP server. It serves the API and both frontends from one host:

| Path | Serves |
|---|---|
| `/api/tracking/...` | customer API (public) |
| `/api/admin/...` | admin API (session cookie, `Path=/api/admin`) |
| `/admin/...` | admin panel SPA (`/admin` redirects to `/admin/`) |
| everything else | customer site SPA |

It uses PostgreSQL when `DB_ADDR` is set, and an in-memory mock store otherwise.

## Deployment

```sh
./scripts/build.sh                      # builds both frontends, embeds them, writes server/bin/brainlog-server
APP_ENV=production ./server/bin/brainlog-server   # run from server/ to pick up .env.production, or pass env vars
```

Or use the container (repo root): `docker build -t brainlog .`, then
`docker run -p 8080:8080 -e DB_ADDR=... brainlog`.

- The image contains no `.env` file, so pass the config as environment variables (`DB_ADDR`, `ADDR`, ...).
- **Put HTTPS in front of it** (your platform's load balancer, or Caddy/nginx). `COOKIE_SECURE` defaults to
  `true`, so the admin login only works over HTTPS.
- **Caching:** files under `assets/` are content-hashed and cached for a year. `index.html` is revalidated
  on every load, so a new deploy takes effect immediately.
- A plain `go build` (without the script) still compiles. The frontends then answer 503
  "frontend not built", while the API works normally.

### Render

`render.yaml` (repo root) is a Render Blueprint for one Docker web service in `singapore`, the same AWS
region as the Neon database. In the Render dashboard, choose **New → Blueprint**, pick the GitHub repo, and paste
the `DB_ADDR` value when prompted.

- Render injects `PORT`. The server listens on it when `ADDR` is unset, so **don't set `ADDR` on Render**.
- Render terminates HTTPS; keep `COOKIE_SECURE=true`. Health check: `GET /healthz`.
- `autoDeployTrigger: checksPass`: a push to master deploys only after CI passes.
- Migrations run on startup. Nothing else needs to run before a deploy.

## Developing the frontends against this server

By default both Vite dev servers use MSW mocks. To use the real server instead:

```sh
cd server && go run .                                     # :8080, in-memory store + mock data
cd customer-facing-client && VITE_USE_MOCKS=false bun run dev   # http://localhost:5173/
cd admin-panel-client && VITE_USE_MOCKS=false bun run dev       # http://localhost:5173/admin/ (or next free port)
```

Vite proxies `/api` to `http://localhost:8080`.

## Run

```sh
cd server

# Local dev: in-memory store, seeded with mock data (see devnote/server-mock.md)
cp .env.example .env
go run .

# Production: loads .env.production (DB_ADDR, ADDR), uses PostgreSQL, runs migrations on start
APP_ENV=production go run .
```

**Config precedence:** real environment variables, then `.env.<APP_ENV>`, then `.env`. `APP_ENV` defaults to
`development`. All `.env*` files except `.env.example` are git-ignored, because they hold the database password.

| Env var | Default | Meaning |
|---|---|---|
| `APP_ENV` | `development` | Which `.env.<APP_ENV>` file to load |
| `ADDR` | `:$PORT`, else `:8080` | Listen address. Leave unset on Render (it sets `PORT`). |
| `PORT` | — | Set by Render and similar platforms; used when `ADDR` is unset |
| `DB_ADDR` | — | PostgreSQL URL, e.g. `postgresql://user:pass@host/db?sslmode=require`. Never logged. |
| `STORE` | auto | `postgres` if `DB_ADDR` is set, otherwise `memory`. Set it to force one. |
| `COOKIE_SECURE` | `true` | `Secure` flag on the session cookie; must be `false` over plain HTTP |
| `SEED_MOCK_DATA` | `true` for memory, **`false` for postgres** | Load the mock dataset. It is skipped if already present. **Never enable it on a real database**: the mock accounts have published passwords. |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |

## Creating accounts from the command line

A new database has no accounts, and the createAccount endpoint needs an admin session. To bootstrap one:

```sh
printf '%s\n' 'the-password' | APP_ENV=production go run . create-account -name "Full Name" -email you@example.com -role admin
```

- This applies pending migrations first, so it also works on an empty database.
- The password comes from stdin, never from a flag, so it doesn't end up in the process list.
- It uses the same validation as the API, and refuses to run against the memory store.

## Layout

```
main.go                      config, store selection, seed, HTTP server, graceful shutdown
internal/api/                routes, auth middleware, validation, JSON wire shapes (dto.go)
internal/store/store.go      Store interface: the only thing handlers talk to
internal/store/pgstore/      PostgreSQL Store (pgx) + embedded migrations (migrations/NNN_*.sql)
internal/store/memstore/     in-memory Store (the mock database)
internal/store/storetest/    contract tests every Store must pass
internal/seed/               mock dataset (mirrors admin-panel-client/src/mocks/data.ts)
internal/model/              domain types
internal/apitime/            yyyy-mm-ddThh:mm:ss.xxxZ format/parse
internal/password/           Argon2id (19 MiB, t=2, p=1), PHC-encoded
internal/ids/                UUIDv7 and 128-bit session ids
internal/textnorm/           case/accent folding for list filters
internal/config/             env + .env files
internal/web/                serves the embedded frontend builds (dist/customer, dist/admin)
```

## Migrations

SQL files in `internal/store/pgstore/migrations/` are embedded in the binary and applied at startup, in
version order. Each is recorded in `schema_migrations`. All pending migrations run in one transaction under a
transaction-scoped advisory lock, so this is safe with several instances starting at once and behind
transaction-mode poolers (e.g. Neon's `-pooler` endpoint). **Never edit an applied migration**; add a new file instead.

## Test

```sh
go test -race ./...    # PostgreSQL tests skip without TEST_DATABASE_URL

# PostgreSQL tests: use a DISPOSABLE database (each test makes and drops its own schema)
docker run -d --rm --name pgtest -e POSTGRES_PASSWORD=postgres -p 55432:5432 postgres:17-alpine
TEST_DATABASE_URL='postgresql://postgres:postgres@localhost:55432/postgres?sslmode=disable' go test -race ./...
```

The PostgreSQL tests include the shared store contract, plus a differential test. That test loads the seed
into both stores and requires identical results for every sort, direction, filter and page.
