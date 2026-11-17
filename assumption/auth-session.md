# Auth & session

- **Session detection:** the client calls `GET /api/admin/me` (assumed endpoint) and does not read the cookie.
  The result is cached in TanStack Query under `['me']` for 5 minutes. A 401 maps to `null` (logged out), not an error.
- **Route guard:** every route except `/login` lives under the pathless layout `routes/_authed.tsx`. Its
  `beforeLoad` redirects to `/login?redirect=<original url>` when `me` is null.
- **After login** the user goes back to `redirect`, which must be a same-app path starting with `/` and not `//`
  (no open redirect). Otherwise they go to `/tracking`. Visiting `/login` while logged in redirects the same way.
- **Session expiry mid-use:** any query that fails with 401 clears `me` and redirects to `/login` with
  `redirect` set, so the user returns to the same page.
- **Cookie lifetime contradiction:** `Login, account.md` says "A cookie is valid for 7 days", but the Session
  section says `ends_time = start_time + 30 days`. The client does not depend on either. Flagged for the server.
- **Email uniqueness:** the Email section says "no 2 accounts share the same `id`", which looks like a
  copy-paste error. **Assumed** it means email is unique, since login is by email and create returns 409 on a
  duplicate email.
- **Logout** calls `POST /api/admin/logout` (assumed), then clears the whole query cache and goes to `/login`
  even if the call fails.
- **Cross-origin:** axios uses `withCredentials: true` so the cookie is sent if the API is ever on another
  origin. That case would also need CORS with credentials on the server.
- **Role in the UI** comes from `me.role`. The *Accounts* nav link is only rendered for admins, and
  `/accounts` redirects non-admins to `/tracking`. The server must still enforce this (it returns 401).
