# Server mock data

Source: `server/internal/seed/seed.go`.

- **In-memory store** (no `DB_ADDR`): seeded by default. Nothing is persisted, and every restart resets to
  this dataset.
- **PostgreSQL** (`DB_ADDR` set): **not** seeded unless `SEED_MOCK_DATA=true` is set explicitly. When it is,
  the data is inserted once; later starts see the seed admin account and skip seeding. Only do this on a
  disposable or dev database, because these passwords are published below.

**The dataset is the same as the admin panel's MSW mock.** It has the same 4 accounts with the same ids,
emails, passwords and roles, the same 24 trackings, and the same status histories, times, creators and
warnings. The full tables are in [admin-mock.md](admin-mock.md) and are not repeated here.

## Differences from the MSW mock

| | MSW mock (admin client) | Go server |
|---|---|---|
| Status ids | `status-1`, `status-2`, … | UUIDv7 built from each status's `creationTime`, so they are random per seed run |
| Passwords | plaintext in `data.ts` | Argon2id-hashed at startup |
| Session | `localStorage` key | real `HttpOnly` cookie `brainlog_session` that refers to an in-memory session row |
| Session lifetime | never expires | cookie Max-Age 7 days; session row `ends_time` 30 days |
| `error500@…` trick | create returns 500 | not implemented, because the real server only returns 500 on real failures |
| Latency | 300 ms artificial | none (login takes about 50 ms because of Argon2id) |
| Customer endpoint | separate mock with `SomeRandomPort…` data | serves the **admin dataset**, so `BLJ-good-code` shows `Hải Phòng, VN → Singapore, SG` |

## Accounts (summary)

| Email | Password | Role |
|---|---|---|
| `admin@brainlog.test` | `Admin123!` | admin |
| `dung@brainlog.test` | `Dung1234!` | admin |
| `staff@brainlog.test` | `Staff123!` | user |
| `ops@brainlog.test` | `Ops12345!` | user |

`01989e2c-5b10-7d3a-9e77-0f1a2b3c4d99` is a creator id with no account row (a "deleted" account).

## Quick curl session

```sh
curl -s localhost:8080/api/tracking/BLJ-good-code                          # public
curl -s -c jar -H 'Content-Type: application/json' \
     -d '{"email":"admin@brainlog.test","password":"Admin123!"}' localhost:8080/api/admin/login
curl -s -b jar 'localhost:8080/api/admin/tracking?pol=da&pageSize=10'      # Đà Nẵng matches
```
