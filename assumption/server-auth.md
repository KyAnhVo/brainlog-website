# Server: sessions & passwords

- **Passwords:** Argon2id with m = 19 MiB, t = 2, p = 1, a 16-byte random salt and a 32-byte key, stored as a
  PHC string (`$argon2id$v=19$m=19456,t=2,p=1$salt$hash`). Verification reads the parameters from the
  stored string, so they can be raised later without breaking existing hashes.
- **Login for an unknown email still runs one Argon2 verification** against a dummy hash, so response
  time does not reveal which emails exist. Both cases return the same 401 message.
- **Session id:** 128 random bits from `crypto/rand`, sent as 32 hex characters in cookie `brainlog_session`.
  - The cookie is `HttpOnly`, `SameSite=Lax`, `Path=/api/admin` (see [deployment.md](deployment.md)), and `Secure` when `COOKIE_SECURE=true` (the default).
  - The **7 days vs 30 days** contradiction is implemented exactly as written: cookie `Max-Age` = 7 days,
    session row `ends_time` = start + 30 days. A browser therefore stays logged in for 7 days. A copied
    cookie value would work until day 30. *Recommend making both 7 days.*
  - The session row stores the raw id, as the spec's `Session` shape says. *Recommend storing
    SHA-256(id) instead, so a database leak does not expose live sessions.*
- **Login replaces** any session cookie the browser already had (the old session is marked invalid),
  which limits session fixation.
- **Logout** sets `is_valid = false` and expires the cookie. It always returns 200, even without a session.
- **A request is authenticated** when the cookie parses, the session exists, `is_valid` is true, now is
  before `ends_time`, and the account still exists.
- **Account deletion deletes all of that account's sessions**, as the spec requires.
- **CSRF:** `SameSite=Lax` stops cross-site POSTs from carrying the cookie. Requiring
  `Content-Type: application/json` also blocks plain HTML form posts. There is no CSRF token.
- **No rate limiting or lockout on login.** Argon2's cost slows brute force but does not stop it.
  *Recommend per-IP and per-account rate limiting before production.*
- **No CORS headers.** The assumption is that the clients and API are served from one origin, through a
  reverse proxy in production and a Vite proxy in dev.
