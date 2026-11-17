# `/tracking` list

- **Default search on load** (as in `requirements/Client-side, admin panel.md`): `/tracking` runs the default
  search as soon as it opens. No params are sent, so the server's defaults apply (newest last status first,
  page 1, 20 per page). *Clear* empties the URL, which runs the default search again.
- **Filters, sort and page live in the URL search params.** Results can be bookmarked and shared, and they
  survive a reload or the Back button from a detail page. Empty fields are omitted so the server applies
  its defaults.
- **Time filters** use `<input type="datetime-local">` in the viewer's **local** time and are converted to
  UTC `yyyy-mm-ddThh:mm:00.000Z`. They have minute precision; there are no seconds.
- **Client-side validation** copies the server's 400 rules so problems are caught before sending:
  min > max, and times more than 5 minutes in the future. The server stays authoritative; if it returns 400
  (e.g. from a hand-edited URL), the error is shown in an alert.
- **Sort UI:** clicking a sortable column header sorts by it in that field's *default* direction (asc for
  trackingNumber/POL/POD, desc for times). Clicking the active column again flips direction. Changing the
  sort, page size or filters resets to page 1. `sortDirection` is omitted from the URL when it is the default.
- **Sortable columns** are exactly the `sortBy` values. *Container* and *Last Status* are shown but not sortable.
- **Pagination** uses MUI `Pagination` with `pageCount` plus a rows-per-page select (10/20/50/100). MUI's
  `TablePagination` is not used because it needs a total row count, which the API does not return.
- **Row click** opens the detail page. The tracking number is also a real link, so it can be middle-clicked
  or opened in a new tab.
- **Last status display:** a chip in the customer timeline's colours. Red means bad, orange means good with
  warnings (the warnings are in a tooltip), and blue outline means good.
- `creatorId` and `customerId` are not shown in the table, to keep it narrow. They are on the detail page.
