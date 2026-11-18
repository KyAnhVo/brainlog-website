# Server: API behaviour

The server implements every endpoint the admin client uses, including the ones marked **ASSUMED API** in
[api-endpoints.md](api-endpoints.md). Paths and shapes follow that file exactly.

## Routes

| Route | Auth |
|---|---|
| `GET /api/tracking/{trackingNumber}` | public |
| `POST /api/admin/login`, `POST /api/admin/logout` | none |
| `GET /api/admin/me` | session |
| `GET /api/admin/tracking`, `GET /api/admin/tracking/{n}`, `POST /api/admin/tracking/{n}/status` | session (any role) |
| `GET /api/admin/account/{id}` | session (any role) |
| `POST /api/admin/createAccount`, `POST /api/admin/deleteAccount` | admin, otherwise **401** as the spec says |

- **Any role may add statuses and view all trackings.** The spec restricts only `/accounts` to admins.
- **No endpoint creates trackings.** No client page needs one and the spec does not define one. Trackings come
  from the seed for now.
- **Customer 404 message:** "No shipment found for this tracking code", the same text as the customer mock.
- Unknown `/api/...` paths return a JSON `ApiError` 404, so clients always get an `ApiError`. A wrong
  method on a known path gets Go's default plain-text 405.

## Requests

- POST bodies must be `Content-Type: application/json` (otherwise **415**) and at most 64 KiB (otherwise **413**).
  They must hold exactly one JSON object with **no unknown fields**. Wrong types, malformed JSON and
  extra fields return 400. This is the "any data type mistake is a bad query" rule applied to bodies.
- **List query parameters:** unknown parameters are ignored. A known parameter given twice (`?page=1&page=2`)
  returns 400. An empty value (`?page=`) is a type error and returns 400, except for the three text filters,
  where an empty value is the default `""`.
- **`page` larger than int32** returns 400 (out of range).
- **`pageCount < page`** is checked after counting matches. That costs a store call, but the spec defines it
  in terms of the result size.

## Validation (status create)

- `occurringTime`: exact wire format, and no later than *now + 5 min* (the same skew buffer as the list).
- `location`: trimmed, nonempty, at most 200 characters.
- good: `status` ∈ `GOOD_STATUS_NAMES`. `warnings` is optional (missing or `null` means `[]`), and each one is
  trimmed, nonempty and at most 500 characters, with at most 20 warnings.
- bad: `status` trimmed, nonempty, at most 500 characters. Sending `warnings` with a bad status returns 400.
- The server sets `id` (UUIDv7), `creatorId` (the session's account) and `creationTime` (now). Clients cannot set them.
- No ordering rule between statuses (see [tracking-detail.md](tracking-detail.md)). A backfilled status
  takes its place in the timeline by `occurringTime`.

## Validation (accounts)

- Email: trimmed, at most 254 characters, a bare address accepted by Go's `net/mail` (no display name).
  Uniqueness is **case-insensitive**, and login email matching is too. The email is stored as typed.
- Password: at least 8 characters (matches the client) and at most 256 bytes, which caps Argon2 input to
  prevent CPU/memory abuse.
- Name: trimmed, 1–100 characters. Role: `admin` | `user`.
- `deleteAccount` with both `email` and `id`, or neither, returns 400.
- **Admins may delete themselves**, including the last admin. Nothing in the spec forbids it, but it can
  lock everyone out of account management. *Recommend forbidding deleting the last admin.*
- Deleting yourself also clears your cookie in the response.

## Times

All times are stored and returned at millisecond precision in UTC. "Current server time" comes from the
server clock, truncated to ms.
