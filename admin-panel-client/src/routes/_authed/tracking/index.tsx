import { Box, Card, CardContent, CardHeader, Divider, LinearProgress, Typography } from '@mui/material';
import { createFileRoute } from '@tanstack/react-router'
import { keepPreviousData, useQuery } from '@tanstack/react-query';

import logo from '@/assets/logo.png';
import ErrorAlert from '@/components/ErrorAlert';
import { api } from '@/lib/api';
import {
  PAGE_SIZES, SORT_FIELDS, defaultSortDirection,
  type PageSize, type SortField, type TrackingListQuery, type TrackingListResponse,
} from '@/lib/tracking';
import TrackingFilters, { type Filters } from './-trackingFilters.tsx'
import TrackingTable from './-trackingTable.tsx'

// Filters live in the URL so a search can be bookmarked, shared, and survives reloads.
// Unknown enum values are dropped; time strings are passed through for the server to validate.
function validateSearch(search: Record<string, unknown>): TrackingListQuery {
  const str = (v: unknown) => (typeof v === 'string' && v !== '' ? v : undefined)
  const int = (v: unknown) => (typeof v === 'number' && Number.isInteger(v) ? v : undefined)
  const sortBy = SORT_FIELDS.find((f) => f === search.sortBy)
  const pageSize = PAGE_SIZES.find((s) => s === search.pageSize)
  return {
    trackingNumber: str(search.trackingNumber),
    pol: str(search.pol),
    pod: str(search.pod),
    creationTimeMin: str(search.creationTimeMin),
    creationTimeMax: str(search.creationTimeMax),
    lastStatusOccurringTimeMin: str(search.lastStatusOccurringTimeMin),
    lastStatusOccurringTimeMax: str(search.lastStatusOccurringTimeMax),
    sortBy,
    sortDirection: search.sortDirection === 'asc' || search.sortDirection === 'desc' ? search.sortDirection : undefined,
    page: int(search.page),
    pageSize,
  }
}

export const Route = createFileRoute('/_authed/tracking/')({
  validateSearch,
  component: RouteComponent,
})

async function fetchTrackings(query: TrackingListQuery, signal: AbortSignal): Promise<TrackingListResponse> {
  const { data } = await api.get<TrackingListResponse>('/admin/tracking', { params: query, signal });
  return data;
}

function RouteComponent() {
  const search = Route.useSearch();
  const navigate = Route.useNavigate();

  // Runs immediately: with no params in the URL the server applies its defaults.
  const { data, error, isFetching } = useQuery({
    queryKey: ['admin-trackings', search],
    queryFn: ({ signal }) => fetchTrackings(search, signal),
    placeholderData: keepPreviousData,
    retry: false,
  });

  const sortBy: SortField = search.sortBy ?? 'lastStatusOccurringTime';
  const sortDirection = search.sortDirection ?? defaultSortDirection(sortBy);

  const filters: Filters = {
    trackingNumber: search.trackingNumber,
    pol: search.pol,
    pod: search.pod,
    creationTimeMin: search.creationTimeMin,
    creationTimeMax: search.creationTimeMax,
    lastStatusOccurringTimeMin: search.lastStatusOccurringTimeMin,
    lastStatusOccurringTimeMax: search.lastStatusOccurringTimeMax,
  }

  const handleSearch = (f: Filters) => {
    navigate({ search: (prev) => ({ ...prev, ...f, page: 1, pageSize: prev.pageSize ?? 20 }) });
  }

  const handleSort = (field: SortField) => {
    navigate({
      search: (prev) => ({
        ...prev,
        sortBy: field,
        // Clicking the active column flips it; a new column starts at its default direction.
        sortDirection: field === sortBy ? (sortDirection === 'asc' ? 'desc' : 'asc') : undefined,
        page: 1,
      }),
    });
  }

  const handlePageSize = (pageSize: PageSize) => {
    navigate({ search: (prev) => ({ ...prev, pageSize, page: 1 }) });
  }

  return (
    <Box sx={{
      minHeight: '100vh',
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'flex-start'
    }}>
      <Card variant='outlined' sx={{
        padding: '1vh',
        margin: '5vh',
        width: '100%',
        maxWidth: 1400,
      }}>
        <CardHeader
          title={
            <Typography
              component="h1"
              sx={{
                fontSize: "40px",
              }}
            >
              Trackings
            </Typography>
          }
          avatar={
            <Box component="img" src={logo} sx={{
              width: 72,
              height: 72,
            }} />
          }
        />
        <CardContent>
          <TrackingFilters
            key={JSON.stringify(filters)}
            initial={filters}
            onSearch={handleSearch}
            onClear={() => navigate({ search: {} })}
          />
          <Divider variant="fullWidth" sx={{ my: 2 }} />
          <Box sx={{ minHeight: 4, mb: 1 }}>{isFetching && <LinearProgress />}</Box>
          {error != null ? (
            <ErrorAlert error={error} />
          ) : data ? (
            <TrackingTable
              data={data}
              sortBy={sortBy}
              sortDirection={sortDirection}
              onSort={handleSort}
              onPage={(page) => navigate({ search: (prev) => ({ ...prev, page }) })}
              onPageSize={handlePageSize}
            />
          ) : null}
        </CardContent>
      </Card>
    </Box>
  )
}
