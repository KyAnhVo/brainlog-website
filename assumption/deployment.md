# Deployment layout

**Decision (from the user):** one Go server serves everything, and the host is split by path:

| Path | Serves |
|---|---|
| `/api/...` | API. All endpoints were already `/api`-prefixed: customer `/api/tracking/...`, admin `/api/admin/...`. |
| `/admin/...` | admin panel |
| `/...` | customer site |

Consequences:

- **Admin URLs change.** The requirements list `/login`, `/tracking`, `/tracking/$trackingId` and `/accounts`
  for the admin panel. In the browser they are now `/admin/login`, `/admin/tracking`, … The admin app is built
  with Vite `base: '/admin/'` and TanStack Router `basepath: '/admin/'`. Inside the app, routes and links are
  unchanged (`/tracking`, `/login`, …), and the login `redirect` parameter holds the in-app path.
- **The customer site and admin panel share one origin.** There is no CORS. The session cookie works with
  `SameSite=Lax`.
- **Session cookie `Path=/api/admin`** (was `/`), so it is never sent to the customer site, static files, or
  the customer API.
- **Same-origin also means shared browser storage.** `localStorage` is per origin, not per path, so the admin
  panel and customer site can read each other's storage. Neither app stores anything sensitive there.
  The admin session is in an `HttpOnly` cookie that JavaScript cannot read. If the customer site ever gets
  untrusted third-party scripts, move the admin panel to its own subdomain.
- **Unknown paths under `/admin/` or `/` without a file extension** return that app's `index.html`, so client
  routes work on reload. A missing file *with* an extension returns 404 (never HTML for a JS request).
  `mockServiceWorker.js` is never served.
- **Security headers on HTML:** CSP (`script-src 'self'`; styles also allow `'unsafe-inline'` for MUI/emotion
  and Google Fonts; fonts from `fonts.gstatic.com`), `X-Frame-Options: DENY`, `nosniff`, and
  `Referrer-Policy: strict-origin-when-cross-origin`. If something new is loaded from another origin (for example
  analytics), the CSP in `server/internal/web/web.go` must be extended.
- **TLS is terminated in front of the server** (platform load balancer or reverse proxy). The server speaks plain HTTP.
- **Builds:** `scripts/build.sh` (local) or the root `Dockerfile` (multi-stage: bun → go → distroless,
  nonroot). The Dockerfile has **not been built in this session** (no Docker daemon available). The build
  script has been run and the resulting binary smoke-tested.
- **The Go version in the Dockerfile** is `golang:1.27`, matching `go.mod` (`go 1.27.3`).
- **Dev:** mocks remain the default. `VITE_USE_MOCKS=false` switches a client to the real server through Vite's
  `/api` proxy (`localhost:8080`).
