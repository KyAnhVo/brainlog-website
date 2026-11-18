# Server: storage (PostgreSQL + in-memory mock)

**Change of plan:** the requirements and earlier notes said MySQL. The production `DB_ADDR` in
`server/.env.production` is a **PostgreSQL** URL (Neon), so the server now has a PostgreSQL store. MySQL is
not planned any more. The MySQL driver was never imported and is no longer in `go.mod`.

- Handlers depend only on `store.Store`. There are two implementations:
  - `pgstore`, used when `DB_ADDR` is set;
  - `memstore`, used otherwise (local dev and tests).

  Both must pass the shared contract suite in `storetest`.
- **The connection string is used exactly as given** (`sslmode`, `channel_binding` are handled by pgx 5.13).
  It is never logged, and parse errors are replaced with a generic message because pgx errors can echo
  the password.
- **Startup fails fast** if the database is unreachable (15-second ping) or a migration fails.
- **Schema** (`pgstore/migrations/001_init.sql`):
  - Ids are `text` (UUIDv7 strings), so a malformed id in a URL is just "not found" and never a cast error.
  - Times are `timestamptz` holding UTC values at millisecond precision.
  - Email uniqueness is a unique index on `lower(email)` (case-insensitive).
  - `sessions.user_id` has `ON DELETE CASCADE`: deleting an account deletes its sessions.
  - `creator_id` columns have **no foreign key**, because trackings and statuses must outlive the account
    that created them.
  - `physicalDetails` is stored as nullable columns, with CHECKs enforcing "all or nothing".
  - `warnings` is `text[]`: never NULL for good statuses, always NULL for bad ones (CHECK enforced).
- **The last status is denormalised** onto `trackings` (`last_status_id`, occurring and creation time). It is
  recomputed in the same transaction as every status insert, with the tracking row locked
  (`SELECT … FOR UPDATE`). The list endpoint filters and sorts on these columns without a subquery per row.
- **Accent- and case-insensitive matching:** the app stores folded copies (`*_fold` columns, computed with
  `textnorm.Fold`) and folds the query the same way. Matching uses `strpos` (contains) and `starts_with`
  (prefix), so `%`/`_` in user input need no escaping. This avoids relying on database collations, which do
  not all treat `đ` as `d`. *If the fold rules ever change, a migration must recompute the `*_fold` columns.*
- **Sorting** compares folded strings with `COLLATE "C"` (byte order). That is the same as memstore's code-point
  order, so both stores return identical pages. A differential test checks this when a test database is available.
- **The list page and the total count** are read in one `REPEATABLE READ` read-only transaction, so they agree.
- **Seeding:** off by default for Postgres. When it is enabled, it is skipped if the seed admin account already
  exists, so restarts don't fail.
- **Prepared statements behind Neon's pooler:** pgx caches named prepared statements by default. Neon's
  PgBouncer supports them (protocol-level prepared statements). If you ever see `prepared statement … does not
  exist` errors behind a different pooler, use the direct (non-`-pooler`) endpoint, or switch pgx to
  `QueryExecModeCacheDescribe`.
- **Untested against a live PostgreSQL in this session:** the Docker daemon was not running here, and the
  production database was deliberately not touched. Run the PostgreSQL tests (see `server/README.md`)
  before deploying.
