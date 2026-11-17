import {
  Box, FormControl, InputLabel, MenuItem, Pagination, Select, Table, TableBody, TableCell, TableContainer,
  TableHead, TableRow, TableSortLabel, Typography,
} from '@mui/material';
import { Link, useNavigate } from '@tanstack/react-router';
import StatusChip from '@/components/StatusChip';
import { formatTime } from '@/lib/time';
import {
  PAGE_SIZES, defaultSortDirection,
  type PageSize, type SortDirection, type SortField, type TrackingListResponse,
} from '@/lib/tracking';

type Column = {
  label: string,
  sortField?: SortField,
}

const COLUMNS: Column[] = [
  { label: 'Tracking Number', sortField: 'trackingNumber' },
  { label: 'Container' },
  { label: 'POL', sortField: 'pol' },
  { label: 'POD', sortField: 'pod' },
  { label: 'Last Status' },
  { label: 'Last Status Time', sortField: 'lastStatusOccurringTime' },
  { label: 'Created', sortField: 'creationTime' },
]

export default function TrackingTable({
  data,
  sortBy,
  sortDirection,
  onSort,
  onPage,
  onPageSize,
}: {
  data: TrackingListResponse,
  sortBy: SortField,
  sortDirection: SortDirection,
  onSort: (field: SortField) => void,
  onPage: (page: number) => void,
  onPageSize: (pageSize: PageSize) => void,
}) {
  const navigate = useNavigate();

  return (
    <>
      <TableContainer>
        <Table size="small">
          <TableHead>
            <TableRow>
              {COLUMNS.map((col) => (
                <TableCell key={col.label} sx={{ fontWeight: 600, whiteSpace: 'nowrap' }}>
                  {col.sortField ? (
                    <TableSortLabel
                      active={sortBy === col.sortField}
                      direction={sortBy === col.sortField ? sortDirection : defaultSortDirection(col.sortField)}
                      onClick={() => onSort(col.sortField!)}
                    >
                      {col.label}
                    </TableSortLabel>
                  ) : col.label}
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {data.trackings.length === 0 ? (
              <TableRow>
                <TableCell colSpan={COLUMNS.length}>
                  <Typography color="text.secondary" sx={{ textAlign: 'center', py: 3 }}>
                    No trackings match these filters.
                  </Typography>
                </TableCell>
              </TableRow>
            ) : data.trackings.map((t) => (
              <TableRow
                key={t.trackingNumber}
                hover
                sx={{ cursor: 'pointer' }}
                onClick={() => navigate({ to: '/tracking/$trackingNumber', params: { trackingNumber: t.trackingNumber } })}
              >
                <TableCell>
                  <Link
                    to="/tracking/$trackingNumber"
                    params={{ trackingNumber: t.trackingNumber }}
                    onClick={(e) => e.stopPropagation()}
                  >
                    {t.trackingNumber}
                  </Link>
                </TableCell>
                <TableCell>{t.cont ?? '—'}</TableCell>
                <TableCell>{t.pol}</TableCell>
                <TableCell>{t.pod}</TableCell>
                <TableCell><StatusChip status={t.lastStatus} /></TableCell>
                <TableCell sx={{ whiteSpace: 'nowrap' }}>{formatTime(t.lastStatus.occurringTime)}</TableCell>
                <TableCell sx={{ whiteSpace: 'nowrap' }}>{formatTime(t.creationTime)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 2, mt: 2, flexWrap: 'wrap' }}>
        <FormControl size="small" sx={{ minWidth: 120 }}>
          <InputLabel id="page-size-label">Rows per page</InputLabel>
          <Select
            labelId="page-size-label"
            label="Rows per page"
            value={data.pageSize}
            onChange={(e) => onPageSize(Number(e.target.value) as PageSize)}
          >
            {PAGE_SIZES.map((size) => <MenuItem key={size} value={size}>{size}</MenuItem>)}
          </Select>
        </FormControl>
        <Pagination
          count={data.pageCount}
          page={data.page}
          onChange={(_, page) => onPage(page)}
          color="primary"
          showFirstButton
          showLastButton
        />
      </Box>
    </>
  )
}
