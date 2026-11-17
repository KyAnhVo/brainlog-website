# Mock (MSW) behaviour vs. a real server

Data and tips are in `devnote/admin-mock.md`.

- **Session:** stored in `localStorage` (`mock-session-account-id`) instead of an HttpOnly cookie, so
  reloads keep you logged in. Sessions never expire in the mock.
- **Data is in memory** and resets on reload: added statuses, created accounts, deleted accounts.
- **Passwords are plaintext** in `mocks/data.ts`. That is acceptable only because it is a mock; the real
  server uses Argon2id.
- **IDs:** account ids look like UUIDv7. New ids get a real ms timestamp and random bits, but not a fully
  RFC 9562-compliant layout. Status ids are `status-N` / `status-new-N`.
- **Accent-insensitive matching** uses Unicode NFD, strips diacritics, and maps `đ/Đ → d` (đ does not
  decompose). A real server/DB collation must also treat `đ` as `d` to match. Sorting by POL/POD/
  trackingNumber uses the same normalised strings, compared by code point; a DB collation may order
  slightly differently.
- **`creationTimeMax`/`lastStatusOccurringTimeMax` defaults** use the browser clock.
- **Validation messages** (`errorMsg`) are invented; only the status codes come from the spec.
- **500 path:** creating an account whose email starts with `error500@` returns 500, so that UI state can be seen.
- The mock tracking numbers `BLJ-good-code*` are shared with the customer mock, but POL/POD/locations
  differ. The two mocks are independent.
