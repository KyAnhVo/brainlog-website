# Customer-facing mock data

Source of truth: `customer-facing-client/src/mocks/handlers.ts`. Mocks run only in `bun run dev`.
Only `GET /api/tracking/:trackingNumber` is mocked, for the three numbers below. Any other tracking number
is not handled by MSW, so the request goes on to the Vite dev server. The dev server normally answers with
its SPA `index.html`, and `lib/api.ts` reports that as *"Unexpected response from server"*.

## Tracking numbers

| Tracking number | Response | What it shows |
|---|---|---|
| `BLJ-good-code` | 200 | Full happy path with all physical details; one warning; two statuses sharing an `occurringTime` |
| `BLJ-good-code-fail-shipment` | 200 | Minimal details (no cont/seal/physical); a warning then a **bad** final status |
| `BLJ-badd-code` | 404 `{ errorCode: 404, errorMsg: 'No shipment found for this tracking code' }` | Error alert |

## `BLJ-good-code`

| Field | Value |
|---|---|
| cont | MSKU1234567 |
| seal | SL998877 |
| pol | SomeRandomPort of Vietnam |
| pod | SomeRandomPort of Singapore |
| physicalDetails | 12 pcs, 840.5 kg, 240 × 120 × 150 cm |

| id | type | status | occurringTime | location | warnings |
|---|---|---|---|---|---|
| status-1 | good | Tracking created | 2026-10-15T09:11:00.000Z | Hanoi, VN | |
| status-2 | good | Customs clearance at POL | 2026-10-15T09:14:00.000Z | Hanoi, VN | |
| status-3 | good | Arrived at POL | 2026-10-20T09:15:00.000Z | Hanoi, VN | |
| status-4 | good | Left POL | 2026-10-21T14:30:00.000Z | Hanoi, VN | |
| status-5 | good | Arrived at POD | 2026-10-23T08:00:00.000Z | Singapore, Singapore | |
| status-6 | good | Customs clearance at POD | 2026-10-24T11:45:00.000Z | Singapore, Singapore | Delayed by customs inspection |
| status-7 | good | Inland trucking to Place of Delivery | 2026-10-24T11:45:00.000Z | Singapore, Singapore | |

## `BLJ-good-code-fail-shipment`

| Field | Value |
|---|---|
| cont / seal / physicalDetails | *(omitted)* |
| pol | SomeRandomPort of Vietnam |
| pod | SomeRandomPort of Singapore |

| id | type | status | occurringTime | location | warnings |
|---|---|---|---|---|---|
| status-8 | good | Tracking created | 2026-10-15T09:11:00.000Z | Hanoi, VN | |
| status-9 | good | Arrived at POL | 2026-10-20T09:15:00.000Z | Hanoi, VN | |
| status-10 | good | Customs clearance at POL | 2026-10-21T10:00:00.000Z | Hanoi, VN | Additional documents requested by customs |
| status-11 | bad | Rejected by customs: prohibited item | 2026-10-22T08:00:00.000Z | Hanoi, VN | |

## Notes / inconsistencies to be aware of

- The admin mock reuses the tracking numbers `BLJ-good-code` and `BLJ-good-code-fail-shipment` with the
  same status names and times, but with **different POL/POD and locations** (realistic port names).
  The two clients' mocks are independent; nothing syncs them.
- In `BLJ-good-code-fail-shipment`, *Arrived at POL* (10-20) comes before *Customs clearance at POL* (10-21);
  in `BLJ-good-code` it is the other way round. Both are allowed, because the spec does not enforce status order.
- `status-6`/`status-7` tie on `occurringTime`. The customer API has no `creationTime`, so the client
  relies on the server's ordering (ties by creationTime ASC).
