# Admin panel mock data

Source of truth: `admin-panel-client/src/mocks/data.ts` (data) and
`admin-panel-client/src/mocks/handlers.ts` (MSW handlers). Mocks run only in `bun run dev`
(`import.meta.env.DEV`); production builds never include them.

## How the mock behaves

- **State is in memory.** Statuses you add and accounts you create or delete last until a page reload.
- **The session is kept in `localStorage`** (key `mock-session-account-id`) so a reload keeps you logged
  in. The real server uses an HttpOnly cookie instead. To force a logout, clear that key or press *Log out*.
- **Every response is delayed 300 ms** so loading states are visible.
- `GET /api/admin/tracking` implements the full spec: filters, case- and accent-insensitive matching,
  sort keys and tie-breakers, pagination, and every 400 rule (see `assumption/tracking-list.md`).
- The mock clock is the browser's `Date.now()`. The mock data runs up to 2026-11-16.

## Accounts

| Name | Email | Password | Role | ID | Created (UTC) |
|---|---|---|---|---|---|
| Alice Nguyen | `admin@brainlog.test` | `Admin123!` | admin | `0199a3f0-1c2e-7a41-9b3e-5d2f8c1a0e01` | 2026-01-05T08:00:00.000Z |
| Bình Trần | `staff@brainlog.test` | `Staff123!` | user | `0199a3f0-4d7b-7c12-8f60-2b9e4a7d1c02` | 2026-02-11T02:30:00.000Z |
| Chloé Martin | `ops@brainlog.test` | `Ops12345!` | user | `0199a3f1-0a93-7e55-a1c4-8e3f6b2d9a03` | 2026-03-20T14:15:00.000Z |
| Dũng Phạm | `dung@brainlog.test` | `Dung1234!` | admin | `0199a3f1-77c2-7f08-b2d1-4c5e9f1a6b04` | 2026-06-01T09:45:00.000Z |

A deleted account, `01989e2c-5b10-7d3a-9e77-0f1a2b3c4d99`, still appears as the creator of
`BLJ-2026-0006` and some of its statuses. Looking up its name returns 404, so the UI shows "Unknown account".

Login email matching is case-insensitive; the password must match exactly.

## Endpoints mocked

| Method & path | Auth | Responses |
|---|---|---|
| `POST /api/admin/login` | — | 200 `LoginSuccessResponse` · 401 |
| `POST /api/admin/logout` | — | 200 `{}` |
| `GET /api/admin/me` | session | 200 `LoginSuccessResponse` · 401 |
| `GET /api/admin/tracking` | session | 200 `TrackingListResponse` · 400 · 401 |
| `GET /api/admin/tracking/:trackingNumber` | session | 200 `AdminTrackingResponse` · 401 · 404 |
| `POST /api/admin/tracking/:trackingNumber/status` | session | 200 `Status` · 400 · 401 · 404 |
| `GET /api/admin/account/:id` | session | 200 `AccountPublic` · 401 · 404 |
| `POST /api/admin/createAccount` | admin | 200 `AccountPublic` · 400 · 401 · 409 · 500 |
| `POST /api/admin/deleteAccount` | admin | 200 `{}` · 400 · 401 · 404 |

## Triggering specific cases

| Want to see | Do this |
|---|---|
| 401 on login | wrong password |
| Redirect to `/login` | log out, then open `/tracking` (you return to it after logging in) |
| Accounts tab hidden / route blocked | log in as `staff@brainlog.test` |
| Empty result | Search with tracking number `zzz` |
| Accent-insensitive search | POL `da` matches **Đà Nẵng**; POL `ho chi` matches **Hồ Chí Minh** |
| Multiple pages | Rows per page = 10 (24 trackings → 3 pages) |
| 400 from list | edit the URL, e.g. `?page=99&pageSize=20` or `?pageSize=15` |
| Tie on occurring time | `BLJ-good-code`: the last two statuses share `occurringTime`; the one entered later is last |
| Backfilled status | `BLJ-2026-0009`: last status was entered 2 days after it occurred |
| Bad last status | `BLJ-good-code-fail-shipment`, `BLJ-2026-0008`, `BLJ-2026-0013`, `BLX-SAMPLE-0021` |
| Bad status in the middle | `BLJ-2026-0004` (typhoon diversion, then arrived with a warning) |
| Multiple warnings | `BLJ-2026-0011` |
| Unknown creator | `BLJ-2026-0006` |
| 404 detail | open `/tracking/does-not-exist` |
| 409 on create | create an account with `ops@brainlog.test` |
| 500 on create | create an account whose email starts with `error500@` |
| 404 on delete | delete `ghost@brainlog.test` |
| Delete yourself | as Alice, delete `admin@brainlog.test` → you are sent to `/login` and that login no longer works until reload |

## Tracking summary

| Tracking number | Cont | Seal | POL | POD | Created (UTC) | Creator | Customer | Last status | Last status time (UTC) |
|---|---|---|---|---|---|---|---|---|---|
| `BLJ-good-code` | MSKU1234567 | SL998877 | Hải Phòng, VN | Singapore, SG | 2026-10-15T09:11:00.000Z | Alice Nguyen | customer-1 | Inland trucking to Place of Delivery | 2026-10-24T11:45:00.000Z |
| `BLJ-good-code-fail-shipment` | — | — | Đà Nẵng, VN | Busan, KR | 2026-10-15T09:11:00.000Z | Alice Nguyen | — | ❌ Rejected by customs: prohibited item | 2026-10-22T08:00:00.000Z |
| `BLJ-2026-0001` | TGHU8821340 | HL220031 | Hồ Chí Minh, VN | Los Angeles, US | 2026-09-02T01:05:00.000Z | Bình Trần | customer-2 | Finished | 2026-10-03T00:00:00.000Z |
| `BLJ-2026-0002` | CMAU5530912 | — | Hồ Chí Minh, VN | Rotterdam, NL | 2026-09-14T03:00:00.000Z | Bình Trần | — | Customs clearance at POD | 2026-10-21T11:00:00.000Z |
| `BLJ-2026-0003` | — | — | Đà Nẵng, VN | Tokyo, JP | 2026-10-01T06:30:00.000Z | Chloé Martin | customer-3 | Finished | 2026-10-09T05:30:00.000Z |
| `BLJ-2026-0004` | OOLU7781203 | OO551200 | Shanghai, CN | Hải Phòng, VN | 2026-10-05T10:00:00.000Z | Alice Nguyen | — | ⚠️ Arrived at POD | 2026-10-14T12:00:00.000Z |
| `BLJ-2026-0005` | — | — | Singapore, SG | Hồ Chí Minh, VN | 2026-10-10T02:15:00.000Z | Dũng Phạm | — | ⚠️ Customs clearance at POD | 2026-10-14T02:00:00.000Z |
| `BLJ-2026-0006` | HLXU2209981 | — | Hải Phòng, VN | Hamburg, DE | 2026-08-20T07:00:00.000Z | *(deleted account)* | customer-4 | Finished | 2026-09-27T14:00:00.000Z |
| `BLJ-2026-0007` | — | — | Port Klang, MY | Đà Nẵng, VN | 2026-10-25T04:40:00.000Z | Bình Trần | — | Arrived at POD | 2026-10-31T10:00:00.000Z |
| `BLJ-2026-0008` | EGLV1100293 | EG093311 | Laem Chabang, TH | Hồ Chí Minh, VN | 2026-10-28T08:20:00.000Z | Chloé Martin | — | ❌ Container damaged during loading | 2026-10-30T07:10:00.000Z |
| `BLJ-2026-0009` | — | — | Hồ Chí Minh, VN | Singapore, SG | 2026-11-02T01:00:00.000Z | Alice Nguyen | customer-1 | Customs clearance at POD | 2026-11-08T03:00:00.000Z |
| `BLJ-2026-0010` | MSCU4432019 | — | Busan, KR | Hải Phòng, VN | 2026-11-03T05:00:00.000Z | Dũng Phạm | — | Left POL | 2026-11-05T08:00:00.000Z |
| `BLJ-2026-0011` | — | — | Hải Phòng, VN | Shanghai, CN | 2026-11-05T03:30:00.000Z | Bình Trần | — | ⚠️ Customs clearance at POL | 2026-11-06T02:00:00.000Z |
| `BLJ-2026-0012` | ZIMU3319028 | ZM112233 | Hồ Chí Minh, VN | Los Angeles, US | 2026-11-06T06:00:00.000Z | Alice Nguyen | customer-2 | Left POL | 2026-11-10T17:00:00.000Z |
| `BLJ-2026-0013` | — | — | Tokyo, JP | Đà Nẵng, VN | 2026-11-08T00:30:00.000Z | Chloé Martin | — | ❌ Shipment held: consignee unreachable | 2026-11-11T08:00:00.000Z |
| `BLJ-2026-0014` | — | — | Đà Nẵng, VN | Singapore, SG | 2026-11-10T09:00:00.000Z | Dũng Phạm | — | Arrived at POD | 2026-11-14T07:00:00.000Z |
| `BLJ-2026-0015` | TCLU6650021 | — | Rotterdam, NL | Hồ Chí Minh, VN | 2026-10-12T12:00:00.000Z | Alice Nguyen | — | Inland trucking to Place of Delivery | 2026-11-14T01:00:00.000Z |
| `BLJ-2026-0016` | — | — | Hải Phòng, VN | Busan, KR | 2026-11-12T02:00:00.000Z | Bình Trần | — | Tracking created | 2026-11-12T02:00:00.000Z |
| `BLJ-2026-0017` | SEGU9001234 | SE776655 | Singapore, SG | Hamburg, DE | 2026-11-13T04:00:00.000Z | Chloé Martin | — | Left POL | 2026-11-15T13:00:00.000Z |
| `BLJ-2026-0018` | — | — | Hồ Chí Minh, VN | Laem Chabang, TH | 2026-11-14T08:00:00.000Z | Dũng Phạm | customer-3 | ⚠️ Ex-work | 2026-11-15T02:00:00.000Z |
| `BLJ-2026-0019` | — | — | Shanghai, CN | Hồ Chí Minh, VN | 2026-11-15T03:00:00.000Z | Alice Nguyen | — | Arrived at POL | 2026-11-16T01:00:00.000Z |
| `BLJ-2026-0020` | — | — | Đà Nẵng, VN | Port Klang, MY | 2026-11-16T07:30:00.000Z | Bình Trần | — | Tracking created | 2026-11-16T07:30:00.000Z |
| `BLX-SAMPLE-0021` | — | — | Los Angeles, US | Hải Phòng, VN | 2026-09-20T18:00:00.000Z | Chloé Martin | — | ❌ Lost in transit: investigation opened | 2026-10-12T05:00:00.000Z |
| `BLX-SAMPLE-0022` | — | — | Busan, KR | Singapore, SG | 2026-10-18T01:00:00.000Z | Dũng Phạm | — | Finished | 2026-10-25T09:00:00.000Z |

## Full status histories

Oldest first. `creationTime` is when the status was entered into the system.

### `BLJ-good-code`

Physical: 12 pcs, 840.5 kg, 240 × 120 × 150 cm

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-1 | good | Tracking created | 2026-10-15T09:11:00.000Z | Hải Phòng, VN |  | Alice Nguyen | 2026-10-15T09:11:00.000Z |
| status-2 | good | Customs clearance at POL | 2026-10-15T09:14:00.000Z | Hải Phòng, VN |  | Alice Nguyen | 2026-10-15T09:29:00.000Z |
| status-3 | good | Arrived at POL | 2026-10-20T09:15:00.000Z | Hải Phòng, VN |  | Bình Trần | 2026-10-20T09:30:00.000Z |
| status-4 | good | Left POL | 2026-10-21T14:30:00.000Z | Hải Phòng, VN |  | Bình Trần | 2026-10-21T14:45:00.000Z |
| status-5 | good | Arrived at POD | 2026-10-23T08:00:00.000Z | Singapore, SG |  | Chloé Martin | 2026-10-23T08:15:00.000Z |
| status-6 | good | Customs clearance at POD | 2026-10-24T11:45:00.000Z | Singapore, SG | Delayed by customs inspection | Chloé Martin | 2026-10-24T11:55:00.000Z |
| status-7 | good | Inland trucking to Place of Delivery | 2026-10-24T11:45:00.000Z | Singapore, SG |  | Chloé Martin | 2026-10-24T12:05:00.000Z |

### `BLJ-good-code-fail-shipment`

Physical: *(none)*

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-8 | good | Tracking created | 2026-10-15T09:11:00.000Z | Đà Nẵng, VN |  | Alice Nguyen | 2026-10-15T09:11:00.000Z |
| status-9 | good | Arrived at POL | 2026-10-20T09:15:00.000Z | Đà Nẵng, VN |  | Bình Trần | 2026-10-20T09:30:00.000Z |
| status-10 | good | Customs clearance at POL | 2026-10-21T10:00:00.000Z | Đà Nẵng, VN | Additional documents requested by customs | Bình Trần | 2026-10-21T10:15:00.000Z |
| status-11 | bad | Rejected by customs: prohibited item | 2026-10-22T08:00:00.000Z | Đà Nẵng, VN |  | Alice Nguyen | 2026-10-22T08:15:00.000Z |

### `BLJ-2026-0001`

Physical: 40 pcs, 18250 kg, 1203 × 235 × 239 cm

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-12 | good | Tracking created | 2026-09-02T01:05:00.000Z | Hồ Chí Minh, VN |  | Bình Trần | 2026-09-02T01:05:00.000Z |
| status-13 | good | Ex-work | 2026-09-03T07:00:00.000Z | Bình Dương, VN |  | Bình Trần | 2026-09-03T07:15:00.000Z |
| status-14 | good | Customs clearance at POL | 2026-09-05T03:20:00.000Z | Hồ Chí Minh, VN |  | Bình Trần | 2026-09-05T03:35:00.000Z |
| status-15 | good | Arrived at POL | 2026-09-05T10:00:00.000Z | Hồ Chí Minh, VN |  | Bình Trần | 2026-09-05T10:15:00.000Z |
| status-16 | good | Left POL | 2026-09-06T22:40:00.000Z | Hồ Chí Minh, VN |  | Bình Trần | 2026-09-06T22:55:00.000Z |
| status-17 | good | Arrived at POT | 2026-09-09T05:10:00.000Z | Kaohsiung, TW |  | Chloé Martin | 2026-09-09T05:25:00.000Z |
| status-18 | good | Left POT | 2026-09-10T18:00:00.000Z | Kaohsiung, TW |  | Chloé Martin | 2026-09-10T18:15:00.000Z |
| status-19 | good | Arrived at POD | 2026-09-28T16:30:00.000Z | Los Angeles, US |  | Chloé Martin | 2026-09-28T16:45:00.000Z |
| status-20 | good | Customs clearance at POD | 2026-09-30T19:00:00.000Z | Los Angeles, US |  | Chloé Martin | 2026-09-30T19:15:00.000Z |
| status-21 | good | Inland trucking to Place of Delivery | 2026-10-01T15:00:00.000Z | Ontario, CA, US |  | Chloé Martin | 2026-10-01T15:15:00.000Z |
| status-22 | good | Arrived at Place of Delivery | 2026-10-02T21:15:00.000Z | Ontario, CA, US |  | Chloé Martin | 2026-10-02T21:30:00.000Z |
| status-23 | good | Finished | 2026-10-03T00:00:00.000Z | Ontario, CA, US |  | Chloé Martin | 2026-10-03T00:15:00.000Z |

### `BLJ-2026-0002`

Physical: 9600 kg, 590 × 235 × 239 cm

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-24 | good | Tracking created | 2026-09-14T03:00:00.000Z | Hồ Chí Minh, VN |  | Bình Trần | 2026-09-14T03:00:00.000Z |
| status-25 | good | Arrived at POL | 2026-09-16T08:00:00.000Z | Hồ Chí Minh, VN |  | Bình Trần | 2026-09-16T08:15:00.000Z |
| status-26 | good | Left POL | 2026-09-18T12:00:00.000Z | Hồ Chí Minh, VN |  | Bình Trần | 2026-09-18T12:15:00.000Z |
| status-27 | good | Arrived at POT | 2026-09-22T06:00:00.000Z | Singapore, SG | Transshipment window extended by 48h | Bình Trần | 2026-09-22T06:15:00.000Z |
| status-28 | good | Left POT | 2026-09-25T09:30:00.000Z | Singapore, SG |  | Bình Trần | 2026-09-25T09:45:00.000Z |
| status-29 | good | Arrived at POD | 2026-10-19T04:45:00.000Z | Rotterdam, NL |  | Chloé Martin | 2026-10-19T05:00:00.000Z |
| status-30 | good | Customs clearance at POD | 2026-10-21T11:00:00.000Z | Rotterdam, NL |  | Chloé Martin | 2026-10-21T11:15:00.000Z |

### `BLJ-2026-0003`

Physical: 3 pcs, 120.25 kg, 80 × 60 × 45 cm

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-31 | good | Tracking created | 2026-10-01T06:30:00.000Z | Đà Nẵng, VN |  | Chloé Martin | 2026-10-01T06:30:00.000Z |
| status-32 | good | Ex-work | 2026-10-02T02:00:00.000Z | Hội An, VN |  | Chloé Martin | 2026-10-02T02:15:00.000Z |
| status-33 | good | Arrived at POL | 2026-10-03T09:00:00.000Z | Đà Nẵng, VN |  | Chloé Martin | 2026-10-03T09:15:00.000Z |
| status-34 | good | Left POL | 2026-10-04T13:00:00.000Z | Đà Nẵng, VN |  | Chloé Martin | 2026-10-04T13:15:00.000Z |
| status-35 | good | Arrived at POD | 2026-10-08T01:00:00.000Z | Tokyo, JP |  | Chloé Martin | 2026-10-08T01:15:00.000Z |
| status-36 | good | Customs clearance at POD | 2026-10-08T07:30:00.000Z | Tokyo, JP |  | Chloé Martin | 2026-10-08T07:45:00.000Z |
| status-37 | good | Arrived at Place of Delivery | 2026-10-09T05:00:00.000Z | Yokohama, JP |  | Chloé Martin | 2026-10-09T05:15:00.000Z |
| status-38 | good | Finished | 2026-10-09T05:30:00.000Z | Yokohama, JP |  | Chloé Martin | 2026-10-09T05:45:00.000Z |

### `BLJ-2026-0004`

Physical: *(none)*

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-39 | good | Tracking created | 2026-10-05T10:00:00.000Z | Shanghai, CN |  | Alice Nguyen | 2026-10-05T10:00:00.000Z |
| status-40 | good | Arrived at POL | 2026-10-06T04:00:00.000Z | Shanghai, CN |  | Alice Nguyen | 2026-10-06T04:15:00.000Z |
| status-41 | good | Left POL | 2026-10-07T20:00:00.000Z | Shanghai, CN |  | Alice Nguyen | 2026-10-07T20:15:00.000Z |
| status-42 | bad | Vessel diverted due to typhoon | 2026-10-09T03:00:00.000Z | South China Sea |  | Dũng Phạm | 2026-10-09T03:15:00.000Z |
| status-43 | good | Arrived at POD | 2026-10-14T12:00:00.000Z | Hải Phòng, VN | Arrived 4 days late | Dũng Phạm | 2026-10-14T12:15:00.000Z |

### `BLJ-2026-0005`

Physical: 1 pcs, 2.5 kg, 30 × 20 × 10 cm

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-44 | good | Tracking created | 2026-10-10T02:15:00.000Z | Singapore, SG |  | Dũng Phạm | 2026-10-10T02:15:00.000Z |
| status-45 | good | Arrived at POL | 2026-10-11T03:00:00.000Z | Singapore, SG |  | Dũng Phạm | 2026-10-11T03:15:00.000Z |
| status-46 | good | Left POL | 2026-10-11T15:00:00.000Z | Singapore, SG |  | Dũng Phạm | 2026-10-11T15:15:00.000Z |
| status-47 | good | Arrived at POD | 2026-10-13T06:00:00.000Z | Hồ Chí Minh, VN |  | Dũng Phạm | 2026-10-13T06:15:00.000Z |
| status-48 | good | Customs clearance at POD | 2026-10-14T02:00:00.000Z | Hồ Chí Minh, VN | Duty unpaid, awaiting consignee | Dũng Phạm | 2026-10-14T02:15:00.000Z |

### `BLJ-2026-0006`

Physical: 220 pcs, 21400 kg, 1203 × 235 × 269 cm

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-49 | good | Tracking created | 2026-08-20T07:00:00.000Z | Hải Phòng, VN |  | *(deleted account)* | 2026-08-20T07:00:00.000Z |
| status-50 | good | Arrived at POL | 2026-08-22T05:00:00.000Z | Hải Phòng, VN |  | *(deleted account)* | 2026-08-22T05:15:00.000Z |
| status-51 | good | Left POL | 2026-08-23T10:00:00.000Z | Hải Phòng, VN |  | *(deleted account)* | 2026-08-23T10:15:00.000Z |
| status-52 | good | Arrived at POT | 2026-08-27T08:00:00.000Z | Singapore, SG |  | *(deleted account)* | 2026-08-27T08:15:00.000Z |
| status-53 | good | Left POT | 2026-08-28T23:00:00.000Z | Singapore, SG |  | *(deleted account)* | 2026-08-28T23:15:00.000Z |
| status-54 | good | Arrived at POD | 2026-09-24T06:00:00.000Z | Hamburg, DE |  | Alice Nguyen | 2026-09-24T06:15:00.000Z |
| status-55 | good | Customs clearance at POD | 2026-09-25T09:00:00.000Z | Hamburg, DE |  | Alice Nguyen | 2026-09-25T09:15:00.000Z |
| status-56 | good | Inland trucking to Place of Delivery | 2026-09-26T05:00:00.000Z | Berlin, DE |  | Alice Nguyen | 2026-09-26T05:15:00.000Z |
| status-57 | good | Arrived at Place of Delivery | 2026-09-27T13:00:00.000Z | Berlin, DE |  | Alice Nguyen | 2026-09-27T13:15:00.000Z |
| status-58 | good | Finished | 2026-09-27T14:00:00.000Z | Berlin, DE |  | Alice Nguyen | 2026-09-27T14:15:00.000Z |

### `BLJ-2026-0007`

Physical: *(none)*

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-59 | good | Tracking created | 2026-10-25T04:40:00.000Z | Port Klang, MY |  | Bình Trần | 2026-10-25T04:40:00.000Z |
| status-60 | good | Arrived at POL | 2026-10-27T02:00:00.000Z | Port Klang, MY |  | Bình Trần | 2026-10-27T02:15:00.000Z |
| status-61 | good | Left POL | 2026-10-28T06:00:00.000Z | Port Klang, MY |  | Bình Trần | 2026-10-28T06:15:00.000Z |
| status-62 | good | Arrived at POD | 2026-10-31T10:00:00.000Z | Đà Nẵng, VN |  | Bình Trần | 2026-10-31T10:15:00.000Z |

### `BLJ-2026-0008`

Physical: 18 pcs, 4410 kg, 590 × 235 × 239 cm

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-63 | good | Tracking created | 2026-10-28T08:20:00.000Z | Laem Chabang, TH |  | Chloé Martin | 2026-10-28T08:20:00.000Z |
| status-64 | good | Customs clearance at POL | 2026-10-29T03:00:00.000Z | Laem Chabang, TH |  | Chloé Martin | 2026-10-29T03:15:00.000Z |
| status-65 | bad | Container damaged during loading | 2026-10-30T07:10:00.000Z | Laem Chabang, TH |  | Chloé Martin | 2026-10-30T07:25:00.000Z |

### `BLJ-2026-0009`

Physical: 6 pcs, 310 kg, 120 × 100 × 90 cm

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-66 | good | Tracking created | 2026-11-02T01:00:00.000Z | Hồ Chí Minh, VN |  | Alice Nguyen | 2026-11-02T01:00:00.000Z |
| status-67 | good | Ex-work | 2026-11-03T02:00:00.000Z | Long An, VN |  | Alice Nguyen | 2026-11-03T02:15:00.000Z |
| status-68 | good | Arrived at POL | 2026-11-04T04:30:00.000Z | Hồ Chí Minh, VN |  | Bình Trần | 2026-11-04T04:45:00.000Z |
| status-69 | good | Left POL | 2026-11-05T11:00:00.000Z | Hồ Chí Minh, VN |  | Bình Trần | 2026-11-05T11:15:00.000Z |
| status-70 | good | Arrived at POD | 2026-11-07T09:00:00.000Z | Singapore, SG |  | Bình Trần | 2026-11-07T09:15:00.000Z |
| status-71 | good | Customs clearance at POD | 2026-11-08T03:00:00.000Z | Singapore, SG |  | Chloé Martin | 2026-11-10T03:00:00.000Z |

### `BLJ-2026-0010`

Physical: *(none)*

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-72 | good | Tracking created | 2026-11-03T05:00:00.000Z | Busan, KR |  | Dũng Phạm | 2026-11-03T05:00:00.000Z |
| status-73 | good | Arrived at POL | 2026-11-04T01:00:00.000Z | Busan, KR |  | Dũng Phạm | 2026-11-04T01:15:00.000Z |
| status-74 | good | Left POL | 2026-11-05T08:00:00.000Z | Busan, KR |  | Dũng Phạm | 2026-11-05T08:15:00.000Z |

### `BLJ-2026-0011`

Physical: 1500 kg, 300 × 200 × 180 cm

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-75 | good | Tracking created | 2026-11-05T03:30:00.000Z | Hải Phòng, VN |  | Bình Trần | 2026-11-05T03:30:00.000Z |
| status-76 | good | Customs clearance at POL | 2026-11-06T02:00:00.000Z | Hải Phòng, VN | Inspection selected (red channel); Missing packing list | Bình Trần | 2026-11-06T02:15:00.000Z |

### `BLJ-2026-0012`

Physical: 960 pcs, 19800 kg, 1203 × 235 × 269 cm

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-77 | good | Tracking created | 2026-11-06T06:00:00.000Z | Hồ Chí Minh, VN |  | Alice Nguyen | 2026-11-06T06:00:00.000Z |
| status-78 | good | Ex-work | 2026-11-07T01:00:00.000Z | Đồng Nai, VN |  | Alice Nguyen | 2026-11-07T01:15:00.000Z |
| status-79 | good | Arrived at POL | 2026-11-08T08:00:00.000Z | Hồ Chí Minh, VN |  | Alice Nguyen | 2026-11-08T08:15:00.000Z |
| status-80 | good | Left POL | 2026-11-10T17:00:00.000Z | Hồ Chí Minh, VN |  | Alice Nguyen | 2026-11-10T17:15:00.000Z |

### `BLJ-2026-0013`

Physical: *(none)*

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-81 | good | Tracking created | 2026-11-08T00:30:00.000Z | Tokyo, JP |  | Chloé Martin | 2026-11-08T00:30:00.000Z |
| status-82 | good | Arrived at POL | 2026-11-09T03:00:00.000Z | Tokyo, JP |  | Chloé Martin | 2026-11-09T03:15:00.000Z |
| status-83 | bad | Shipment held: consignee unreachable | 2026-11-11T08:00:00.000Z | Tokyo, JP |  | Chloé Martin | 2026-11-11T08:15:00.000Z |

### `BLJ-2026-0014`

Physical: 2 pcs, 48 kg, 60 × 40 × 40 cm

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-84 | good | Tracking created | 2026-11-10T09:00:00.000Z | Đà Nẵng, VN |  | Dũng Phạm | 2026-11-10T09:00:00.000Z |
| status-85 | good | Arrived at POL | 2026-11-11T02:00:00.000Z | Đà Nẵng, VN |  | Dũng Phạm | 2026-11-11T02:15:00.000Z |
| status-86 | good | Left POL | 2026-11-12T05:00:00.000Z | Đà Nẵng, VN |  | Dũng Phạm | 2026-11-12T05:15:00.000Z |
| status-87 | good | Arrived at POD | 2026-11-14T07:00:00.000Z | Singapore, SG |  | Dũng Phạm | 2026-11-14T07:15:00.000Z |

### `BLJ-2026-0015`

Physical: *(none)*

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-88 | good | Tracking created | 2026-10-12T12:00:00.000Z | Rotterdam, NL |  | Alice Nguyen | 2026-10-12T12:00:00.000Z |
| status-89 | good | Arrived at POL | 2026-10-14T08:00:00.000Z | Rotterdam, NL |  | Alice Nguyen | 2026-10-14T08:15:00.000Z |
| status-90 | good | Left POL | 2026-10-15T22:00:00.000Z | Rotterdam, NL |  | Alice Nguyen | 2026-10-15T22:15:00.000Z |
| status-91 | good | Arrived at POT | 2026-11-01T04:00:00.000Z | Colombo, LK |  | Alice Nguyen | 2026-11-01T04:15:00.000Z |
| status-92 | good | Left POT | 2026-11-03T10:00:00.000Z | Colombo, LK |  | Alice Nguyen | 2026-11-03T10:15:00.000Z |
| status-93 | good | Arrived at POD | 2026-11-12T06:00:00.000Z | Hồ Chí Minh, VN |  | Bình Trần | 2026-11-12T06:15:00.000Z |
| status-94 | good | Customs clearance at POD | 2026-11-13T03:00:00.000Z | Hồ Chí Minh, VN |  | Bình Trần | 2026-11-13T03:15:00.000Z |
| status-95 | good | Inland trucking to Place of Delivery | 2026-11-14T01:00:00.000Z | Bình Dương, VN |  | Bình Trần | 2026-11-14T01:15:00.000Z |

### `BLJ-2026-0016`

Physical: *(none)*

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-96 | good | Tracking created | 2026-11-12T02:00:00.000Z | Hải Phòng, VN |  | Bình Trần | 2026-11-12T02:00:00.000Z |

### `BLJ-2026-0017`

Physical: 75 pcs, 12040.75 kg, 1203 × 235 × 239 cm

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-97 | good | Tracking created | 2026-11-13T04:00:00.000Z | Singapore, SG |  | Chloé Martin | 2026-11-13T04:00:00.000Z |
| status-98 | good | Arrived at POL | 2026-11-14T06:00:00.000Z | Singapore, SG |  | Chloé Martin | 2026-11-14T06:15:00.000Z |
| status-99 | good | Left POL | 2026-11-15T13:00:00.000Z | Singapore, SG |  | Chloé Martin | 2026-11-15T13:15:00.000Z |

### `BLJ-2026-0018`

Physical: *(none)*

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-100 | good | Tracking created | 2026-11-14T08:00:00.000Z | Hồ Chí Minh, VN |  | Dũng Phạm | 2026-11-14T08:00:00.000Z |
| status-101 | good | Ex-work | 2026-11-15T02:00:00.000Z | Thủ Đức, VN | Pickup rescheduled once | Dũng Phạm | 2026-11-15T02:15:00.000Z |

### `BLJ-2026-0019`

Physical: 10 pcs, 560 kg, 200 × 120 × 100 cm

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-102 | good | Tracking created | 2026-11-15T03:00:00.000Z | Shanghai, CN |  | Alice Nguyen | 2026-11-15T03:00:00.000Z |
| status-103 | good | Arrived at POL | 2026-11-16T01:00:00.000Z | Shanghai, CN |  | Alice Nguyen | 2026-11-16T01:15:00.000Z |

### `BLJ-2026-0020`

Physical: *(none)*

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-104 | good | Tracking created | 2026-11-16T07:30:00.000Z | Đà Nẵng, VN |  | Bình Trần | 2026-11-16T07:30:00.000Z |

### `BLX-SAMPLE-0021`

Physical: *(none)*

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-105 | good | Tracking created | 2026-09-20T18:00:00.000Z | Los Angeles, US |  | Chloé Martin | 2026-09-20T18:00:00.000Z |
| status-106 | good | Arrived at POL | 2026-09-22T16:00:00.000Z | Los Angeles, US |  | Chloé Martin | 2026-09-22T16:15:00.000Z |
| status-107 | good | Left POL | 2026-09-24T02:00:00.000Z | Los Angeles, US |  | Chloé Martin | 2026-09-24T02:15:00.000Z |
| status-108 | good | Arrived at POT | 2026-10-08T03:00:00.000Z | Kaohsiung, TW |  | Chloé Martin | 2026-10-08T03:15:00.000Z |
| status-109 | bad | Lost in transit: investigation opened | 2026-10-12T05:00:00.000Z | Kaohsiung, TW |  | Dũng Phạm | 2026-10-12T05:15:00.000Z |

### `BLX-SAMPLE-0022`

Physical: *(none)*

| id | type | status | occurringTime | location | warnings | creator | creationTime |
|---|---|---|---|---|---|---|---|
| status-110 | good | Tracking created | 2026-10-18T01:00:00.000Z | Busan, KR |  | Dũng Phạm | 2026-10-18T01:00:00.000Z |
| status-111 | good | Arrived at POL | 2026-10-19T05:00:00.000Z | Busan, KR |  | Dũng Phạm | 2026-10-19T05:15:00.000Z |
| status-112 | good | Left POL | 2026-10-20T09:00:00.000Z | Busan, KR |  | Dũng Phạm | 2026-10-20T09:15:00.000Z |
| status-113 | good | Arrived at POD | 2026-10-24T05:00:00.000Z | Singapore, SG |  | Dũng Phạm | 2026-10-24T05:15:00.000Z |
| status-114 | good | Customs clearance at POD | 2026-10-24T12:00:00.000Z | Singapore, SG |  | Dũng Phạm | 2026-10-24T12:15:00.000Z |
| status-115 | good | Arrived at Place of Delivery | 2026-10-25T08:00:00.000Z | Jurong, SG |  | Dũng Phạm | 2026-10-25T08:15:00.000Z |
| status-116 | good | Finished | 2026-10-25T09:00:00.000Z | Jurong, SG |  | Dũng Phạm | 2026-10-25T09:15:00.000Z |
