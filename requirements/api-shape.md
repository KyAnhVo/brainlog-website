# Error

All errors are of type

```ts
type ApiError = {
	errorCode: number,
	errorMsg: string,
}
```

# Time

 Time is sent/received at UTC time, `RFC 3339` with `Z` suffix and millisecond precision, both in JSON body and query parameters.  The shape MUST BE `yyyy-mm-ddThh:mm:ss.xxxZ`. Even if `xxx == 000`, `000` must still be there.

```typescript
type Time = string;
function isValidTime(s: string): boolean
```

`isValidTime` rejects anything not exactly of that shape, including out-of-range fields that `Date.parse` would roll over (e.g. `2026-02-30T00:00:00.000Z`).
# Shape requirements

## Status (a cargo's status)

```ts
type GoodStatus = {
  statusType: 'good',
  status: GoodStatusName,
  warnings: string[],  // empty array when none, never omitted
}

type BadStatus = {
  statusType: 'bad',
  status: string,  // any nonempty string
}

type StatusCustomer = (GoodStatus | BadStatus) & {
  // metadatas
  id: string,

  // occurence
  occurringTime: Time,
  location: string,
};

type Status = StatusCustomer & {
	// status creator metadata
	creatorId: string,
	creationTime: Time,
}
```

Clients must `switch`/`match` on `statusType`.

The possible values of good status are

```ts
const GOOD_STATUS_NAMES = [
  'Tracking created', 'Ex-work', 'Customs clearance at POL', 'Arrived at POL',
  'Left POL', 'Arrived at POT', 'Left POT', 'Arrived at POD', 'Left POD',
  'Customs clearance at POD', 'Inland trucking to Place of Delivery',
  'Arrived at Place of Delivery', 'Finished',
] as const

type GoodStatusName = typeof GOOD_STATUS_NAMES[number]

const POSSIBLE_STATUSES: ReadonlySet<string> = new Set(GOOD_STATUS_NAMES)
```

All tracking, at creation, will have at least `Tracking created` status.
## Tracking Metadata

```ts
type TrackingBase = {
	trackingNumber: string,
	cont?: string,
	seal?: string,
	pol: string,
	pod: string,
	physicalDetails?: {
		pcsCount?: number,
		weightKg: number,
		dimensions: {
			lengthCm: number,
			widthCm: number,
			heightCm: number,
		},
	},
}

type Tracking = TrackingBase & {
	creatorId: string,
	creationTime: Time,
	lastStatus: Status,
	customerId?: string,
}
```

## Note on status

A tracking's last status is the status with the greatest occurring time; ties are broken by the greatest creation time.

Occurring time is always in the past.

Bad status can get any string, nonempty.

# `GET /api/admin/tracking` (admin panel)

## Notes

- Comparisons are case-insensitive and accent-insensitive.

## Query parameters for filter and sorting:

- `trackingNumber`
	- Default: `""`
	- Behavior: all tracking with tracking numbers X s.t. `X.contains(trackingNumber)`
- `pol`
	- Default: `""`
	- Behavior: all tracking with pol X s.t. `X.startsWith(pol)`
- `pod`
	- Default: `""`
	- Behavior: Refer to `pol`.
- `creationTimeMin`
	- Default: `1970-01-01T00:00:00.000Z`, which is the lowest time bound
	- Behavior: all tracking with creation time t s.t. `t >= creationTimeMin`
- `creationTimeMax`
	- Default: current server time
	- Behavior: Refer to `creationTimeMin`.
- `lastStatusOccurringTimeMin`
	- Default: `1970-01-01T00:00:00.000Z`
	- Behavior: all tracking X where `X.lastStatus.occurringTime >= lastStatusOccurringTimeMin`
- `lastStatusOccurringTimeMax`
	- Default: current server time
	- Behavior: Refer to `lastStatusOccurringTimeMin`
- `sortBy`
	- Default: `lastStatusOccurringTime`
	- Possible values: `trackingNumber`, `pol`, `pod`, `lastStatusOccurringTime`, `creationTime`
	- Behavior: defines the `desc` ordering as the key list `sortBy DESC`, `lastStatusOccurringTime DESC`, `lastStatus.creationTime DESC` (not selectable via `sortBy`), `creationTime DESC`, `trackingNumber ASC`. Keys repeating an earlier key are skipped.
- `sortDirection`
	- Possible values: `asc`, `desc`
	- Default: dependent on field: `pol`, `pod`, `trackingNumber` are `asc`, the rest are `desc`
	- Behavior: `desc` uses the ordering defined by `sortBy`; `asc` is its exact reverse (every key, including tiebreakers, flips direction).
- `page`
	- Default: `1`
	- Possible values: integers
	- Behavior: for pagination
- `pageSize`
	- Default: `20`
	- Possible values: `10`, `20`, `50`, `100`

## Response Type, good

Response type is of shape

```ts
type TrackingListResponse = {
	trackings: Tracking[],
	page: number,      // integer
	pageCount: number, // integer
	pageSize: number,  // integer
}
```

`page` field is the currently queried page.

`pageCount` is the number of pages with the given `pageSize`. 

`pageSize` is the size of the page queried. Let `T = {tracking | tracking satisfies query}`. Then `(pageCount - 1) * pageSize < |T| <= pageCount * pageSize`.

if there is nothing in `trackings`, it is still considered a good query and `trackings == []`.
## Response Type, bad query

- `{anything}Min > {anything}Max` is considered bad query (400).
- `sortBy` not in the listed fields is considered bad query (400).
- `pageSize` not in the given list is bad query (400).
- `page < 1` is a bad query (400).
- `pageCount` < `page` is considered a bad query unless `pageCount == 0 && page == 1` (400).
- Any time dependent values greater than current server time + `acceptedBuffer` is considered bad query (400), where `acceptedBuffer = 5 minutes` (clock skew allowance).
- Any query `<queryField>=<queryValue` such that `queryValue not in queryField.domain()` is considered bad query (400).
- Any data type mistake is a bad query (400).

# `GET /api/tracking/:trackingNumber` (customer-facing)

### 200 OK

```ts
type TrackingResponse = {
  details: TrackingBase,  // last status is the last entry of `statuses`
  statuses: StatusCustomer[], // oldest first: occurringTime ASC, ties by creationTime ASC
}
```

# TODO (unspecified)

- Endpoint to resolve `creatorId` → display name. Every creator field is an id, so the admin panel cannot show creator names without it.
- `GET /api/admin/tracking/:trackingNumber` (admin single-tracking view, full `Status[]` including `creatorId`/`creationTime`). Shape not defined yet.
