# API endpoints (admin)

All paths are relative to `VITE_API_BASE_URL` (`/api`, same as the customer client).

## Path prefix

`Login, account.md` says `POST /admin/login` and `POST /admin/createAccount`, but `api-shape.md` puts admin
data under `/api/admin/...`. **Assumed:** the auth endpoints are under the same `/api` base:
`/api/admin/login` and `/api/admin/createAccount`.

## Specified endpoints, as used

| Endpoint | Notes |
|---|---|
| `POST /api/admin/login` | Body `Login`; 200 `LoginSuccessResponse` (snake_case `account_created` kept as written) |
| `POST /api/admin/createAccount` | Body `AccountCreate`. **Assumed** the 200 body is JSON `{ id, name }` (spec only says "200"), so the UI can show the new ID. |
| `GET /api/admin/tracking` | Exactly as specified |

## ASSUMED API (not in the requirements)

| Endpoint | Shape | Why |
|---|---|---|
| `GET /api/admin/me` | 200 `LoginSuccessResponse` / 401 | The session cookie should be HttpOnly, so JS cannot read it. The client needs a way to ask "am I logged in, and what is my role?" after a reload. |
| `POST /api/admin/logout` | 200 `{}` | Not specified; a logout button is expected. The server should set `is_valid=false` for the session and clear the cookie. |
| `POST /api/admin/deleteAccount` | Body `{ email } \| { id }`; 200 `{}` / 401 / 404 | The spec gives the body and responses but no path. It is named to match `createAccount`. Uses POST rather than DELETE because the spec sends a body. |
| `GET /api/admin/tracking/:trackingNumber` | `{ details: Omit<Tracking,'lastStatus'>, statuses: Status[] }` | Listed as TODO in `api-shape.md`. Mirrors the customer `TrackingResponse`, but with admin fields (`creatorId`, `creationTime`, `customerId`) and full `Status` (with `creatorId`/`creationTime`). Ordering is the same as the customer endpoint. |
| `POST /api/admin/tracking/:trackingNumber/status` | Body `StatusCreate = (GoodStatus \| BadStatus) & { occurringTime, location }`; 200 created `Status` | Needed for the "status updating section". The server assigns `id`, `creatorId` (from the session) and `creationTime`. 400 if `occurringTime` is in the future (5-minute buffer, same as the list endpoint) or the status is invalid. |
| `GET /api/admin/account/:id` | 200 `{ id, name }` / 404 | Listed as TODO ("creatorId → display name"). One request per id, cached forever on the client. A batch endpoint would be better if lists ever show creator names. |

## Success bodies

`lib/api.ts` (copied from the customer client) rejects any 2xx response that is not a JSON object, to catch
SPA fallbacks. **Assumed:** every admin endpoint returns a JSON object on success, `{}` when there is nothing to say.

## 401 vs 403

The spec uses 401 for "logged in but not an admin" on account endpoints. The client handles that: mutations
show "You are not allowed…". However, a 401 from any *query* is treated as "session expired" and redirects to
`/login`. If the server ever returns 401 for a permission problem on a GET, that would log the user out.
Recommend 403 for permission errors.

## Unknown query parameters

**Assumed:** unknown query keys on `GET /api/admin/tracking` are ignored (not 400).
