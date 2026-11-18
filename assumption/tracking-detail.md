# `/tracking/$trackingNumber` detail

- **Route param name:** the requirements say `/tracking/$trackingId`, but trackings have no separate id;
  `trackingNumber` is the key everywhere else. **Used `$trackingNumber`**, matching the customer client.
- **Data source:** assumed `GET /api/admin/tracking/:trackingNumber` (see `api-endpoints.md`).
- **Layout:** same *Item Details* card and MUI Lab `Timeline` as the customer page, oldest first, same dot
  colours. The admin version adds:
  - details: last status, customer ID, created by (name), created at;
  - each status: "Entered by \<name\> · \<creationTime\>".
- **Creator names** come from `GET /api/admin/account/:id` (assumed). A 404 (deleted account) shows
  *"Unknown account"*; the raw id is always in the tooltip.
- **Back button** uses browser history when possible, so the list keeps its filters. On a direct visit it
  links to `/tracking`.

## Status updating section

- **Add only.** Editing or deleting existing statuses is not in the requirements and is not built.
- **Good vs. bad** is a toggle labelled *Normal* / *Problem*:
  - *Normal*: select from `GOOD_STATUS_NAMES` plus an optional list of free-text warnings (blank ones are dropped);
  - *Problem*: free text, required, trimmed nonempty.
- **Defaults:** the suggested good status is the step after the latest good status in `GOOD_STATUS_NAMES`
  order. Location defaults to the latest status's location. Occurred-at defaults to now.
- **Status order is not enforced.** Any good status may be added at any time, statuses may be backfilled,
  and statuses can still be added after *Finished* or a bad status. The spec does not restrict this.
- **Occurred-at must not be in the future** (spec: "Occurring time is always in the past"). The client
  checks against `now`; the mock server allows a 5-minute clock-skew buffer. Occurred-at is **not**
  required to be after the tracking's creation time (not specified).
- **Location is required** (nonempty). The spec types it as `string` without saying it may be empty, and a
  blank location is useless on the customer timeline.
- After a successful add, the detail query and all list queries are invalidated, so the timeline and
  last-status chips refresh.
- **Creating trackings:** see [tracking-create.md](tracking-create.md).
