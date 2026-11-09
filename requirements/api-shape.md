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
# Shape requirements

## Tracking Metadata

Each tracking is associated with its properties:
- Tracking Number
- (Optional) Cont / Seal
- POL
- POD
- (Optional) Physical Details
  - (Optional) pieces count
  - (Optional) weight (kg)
  - (Optional) dimensions (meter)

## Additional information, Admin panel

To show a tracking in the admin panel, we also the following:
- Tracking creator
- Tracking creation time
- Tracking last status
- (Optional) Name/email/phone/id of customer. Or a hyperlink to /customer/$customerId.

## Good status

Each good status must contain the following data:
- Status (`str`)
- Occurring time
- Location
- Warnings
- Creation time (time the status is inputted into the system)
- Name of creator of status

Both creation time of name of status creator are admin panel only

## Bad status

A bad status is when bad things happen to a cargo. It could be that the cargo fell of a ship, customs do not approve of the item, etc.

Each bad (error) status must contain the following data:
- status  (`str`)
- Occurring time
- Location
- Creation time
- Name of creator of status

Both creation time of name of status creator are admin panel only

## Note on status

The possible values of good status is 

```ts
const POSSIBLE_STATUSES = new Set([
  'Tracking created', 'Ex-work', 'Customs clearance at POL', 'Arrived at POL', 'Left POL',
  'Arrived at POT', 'Left POT', 'Arrived at POD', 'Left POD',
  'Customs clearance at POD', 'Inland trucking to Place of Delivery',
  'Arrived at Place of Delivery', 'Finished'
])
```

All tracking, at creation, will have at least `Tracking created` status.

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
	- Default: `Date(0)`, which is the lowest time bound
	- Behavior: all tracking with creation time t s.t. `t >= creationTimeMin`
- `creationTimeMax`
	- Default: `Date.now()`
	- Behavior: Refer to `creationTimeMin`.
- `lastStatusOccurringTimeMin`
	- Default: `Date(0)`
	- Behavior: all tracking X where `X.lastStatus.occurringTime >= lastStatusOccurringTimeMin`
- `lastStatusOccurringTimeMax`
	- Default: `Date.now()`
	- Behavior: Refer to `lastStatusOccurringTimeMin`
- `sortBy`
	- Default: `lastStatusOccurringTime`
	- Possible values: `trackingNumber`, `pol`, `pod`, `lastStatusOccurringTime`, `creationTime`
	- Behavior: Sort by that value. `lastStatusOccurringTime DESC`, then `lastStatus.creationTime DESC` (not in the list), then `creationTime DESC`, then `trackingNumber ASC` will always be the tiebreaker.
- `sortDirection`
	- Possible values: `asc`, `desc`
	- Default: dependent on field: `pol`, `pod`, `trackingNumber` are `asc`, the rest are `desc`
	- Behavior: dictates the sort direction
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
type Response = {
	trackings: Tracking[],
	page: int,
	pageCount: int,
	pageSize: int,
}
```

`page` field is the currently queried page.

`pageCount` is the number of pages with the given `pageSize`. 

`pageSize` is the size of the page queried. Let `T = {tracking | tracking satisfies query}`. Then `(pageCount - 1) * pageSize < |T| <= pageCount * pageSize`.

`trackings` field is of type `Tracking[]`.  Use the above query parameters for sorting. At the very least, `Tracking` objects are defined as following:

```ts
type Tracking = {
	trackingNumber: string,
	pol: string,
	pod: string,
	creationTime: string,
	creator: string,
	lastStatus: {
		status: string,
		type: "good" | "bad",
	},
	lastStatusOccurringTime: string,
	lastStatusCreator: string,
}
```

if there is nothing in `trackings`, it is still considered a good query and `trackings == []`.
## Response Type, bad query

- `{anything}Min > {anything}Max` is considered bad query (400).
- `sortBy` not in the listed fields is considered bad query (400).
- `pageSize` not in the given list is bad query (400).
- `page < 1` is a bad query (400).
- `pageCount` < `page` is considered a bad query unless `pageCount == 0 && page == 1` (400).
- Any time dependent values greater than `Date.now() + acceptedBuffer` is considered bad query (400).
- Any query `<queryField>=<queryValue` such that `queryValue not in queryField.domain()` is considered bad query (400).
- Any data type mistake is a bad query (400).

# `GET /api/tracking/:trackingId` (customer-facing)

### 200 OK

```ts
{
  details: {
    trackingNumber: string,
    contOrSeal?: string,
    pol: string,                  // Port of Loading
    pod: string,                  // Port of Discharge
    physicalDetails?: {
      pieces?: number,
      weightKg?: number,
      dimensionsM?: { length: number, width: number, height: number },
    },
  },
  statuses: (GoodStatus | BadStatus)[],   // ordered by occurringTime, oldest first
}

type GoodStatus = {
  type: 'good',
  status: string,
  occurringTime: string,
  location: string,
  warnings: string[],             // empty array when none, never omitted
}

type BadStatus = {
  type: 'bad',
  status: string,
  occurringTime: string,
  location: string,
}
```

`type` is the discriminator; both variants carry `status`, so clients must switch on `type`.

Optional fields are omitted when absent, not sent as `null`.

### Errors

Any non-2xx response carries a JSON body:

```ts
{ errorCode: number, errorMsg: string }
```

e.g. `404 { errorCode: 404, errorMsg: 'No shipment found for this tracking code' }`.

# `Get /api/admin/tracking/$trackingId` (admin panel)

These exist on every status (good and bad) but must **not** be returned by the customer-facing endpoint:

```ts
{
  creationTime: string,           // when the status was entered into the system
  statusCreator: string,          // name of whoever entered it
}
```

Tracking-level admin fields (creator, creation time, latest status, optional customer link) are listed in the requirements; their endpoint shape is not defined yet.

## Mock tracking codes (customer-facing-client MSW)

| Code | Result |
|---|---|
| `BLJ-good-code` | 200, all detail fields, good statuses only (one with a warning) |
| `BLJ-good-code-fail-shipment` | 200, minimal details, ends in a bad status |
| `BLJ-badd-code` | 404 error body |

