# `/tracking/new`: creating a tracking

Built from `TrackingBase`/`Tracking` in `requirements/api-shape.md`. These decisions were confirmed with the user:

| Question | Decision |
|---|---|
| Who picks the tracking number? | **The server, at random**: `BLJ-XXXX-XXXX`, 8 symbols from a 32-symbol alphabet without 0/O/1/I (40 bits). Numbers are the only key to the **public** customer endpoint, so they must not be sequential or guessable. The mock data's `BLJ-2026-0001` style stays as old data. |
| First status | `Tracking created` (good, no warnings), `occurringTime = creationTime = now`, `location = POL`, `creatorId` = the logged-in user. Created in the same transaction as the tracking. |
| `customerId` | **Not in the form or the API** (no customer entity exists). Sending it returns 400. |
| `cont` / `seal` | Optional free text, trimmed, at most 100 characters. Blank means absent. No ISO 6346 check. |

## ASSUMED API: `POST /api/admin/tracking`

- **Auth:** any logged-in role. The requirements list `/tracking/new` outside the admin-only section.
- **Body:** `Omit<TrackingBase, 'trackingNumber'>`, i.e. `{ cont?, seal?, pol, pod, physicalDetails? }`. Unknown
  fields (including `trackingNumber` and `customerId`) return 400.
- **Validation:**
  - `pol`, `pod`: trimmed, nonempty, at most 200 characters.
  - `physicalDetails`, if present: `weightKg` and all three `dimensions` are required and must be > 0 (sanity cap
    1e9). `pcsCount` is optional and must be an integer ≥ 1 (1.5 is a type error, so 400).
  - Units are fixed (kg, cm), as in the spec.
- **Response 200:** `AdminTrackingResponse` (same as `GET /api/admin/tracking/:trackingNumber`). The client uses
  it to show the detail page without another request.
- **Number collisions** (practically impossible) are retried up to 5 times before a 500.

## UI

- A *New tracking* button on the `/tracking` card header leads to `/admin/tracking/new`.
- The form has route (POL, POD, required), container (cont, seal, optional), and a switch for
  *Add physical details* (weight, pieces optional, L × W × H).
- An info note says the number is generated and "Tracking created" is added at the POL.
- On success it goes to the new tracking's detail page (replacing `/new` in history, so Back returns to the
  list), and the list is refreshed.
- **Not built:** editing or deleting trackings after creation. Not requested.
- **Untested on PostgreSQL:** the create path was tested on the in-memory store and the built binary, not
  against the real database, which I avoided writing test data into. `CreateTracking` is covered by
  the store contract tests when `TEST_DATABASE_URL` is set.
