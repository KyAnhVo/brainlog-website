import { Box, Button, Stack, TextField, Typography } from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import React, { useState } from 'react';
import { fromLocalInput, toLocalInput, type Time } from '@/lib/time';
import type { TrackingListQuery } from '@/lib/tracking';

type Filters = Pick<TrackingListQuery,
  'trackingNumber' | 'pol' | 'pod' | 'creationTimeMin' | 'creationTimeMax'
  | 'lastStatusOccurringTimeMin' | 'lastStatusOccurringTimeMax'>

type FormState = Record<keyof Filters, string>

const ACCEPTED_BUFFER_MS = 5 * 60_000

function toForm(f: Filters): FormState {
  return {
    trackingNumber: f.trackingNumber ?? '',
    pol: f.pol ?? '',
    pod: f.pod ?? '',
    creationTimeMin: toLocalInput(f.creationTimeMin),
    creationTimeMax: toLocalInput(f.creationTimeMax),
    lastStatusOccurringTimeMin: toLocalInput(f.lastStatusOccurringTimeMin),
    lastStatusOccurringTimeMax: toLocalInput(f.lastStatusOccurringTimeMax),
  }
}

// Empty fields are omitted so the server applies its defaults.
function toFilters(s: FormState): Filters {
  const text = (v: string) => (v.trim() === '' ? undefined : v.trim())
  return {
    trackingNumber: text(s.trackingNumber),
    pol: text(s.pol),
    pod: text(s.pod),
    creationTimeMin: fromLocalInput(s.creationTimeMin),
    creationTimeMax: fromLocalInput(s.creationTimeMax),
    lastStatusOccurringTimeMin: fromLocalInput(s.lastStatusOccurringTimeMin),
    lastStatusOccurringTimeMax: fromLocalInput(s.lastStatusOccurringTimeMax),
  }
}

// Mirrors the server's 400 rules so the user gets feedback before a round trip.
function validate(f: Filters): Partial<Record<keyof Filters, string>> {
  const errors: Partial<Record<keyof Filters, string>> = {}
  const limit = Date.now() + ACCEPTED_BUFFER_MS
  const future = (t: Time | undefined) => t != null && Date.parse(t) > limit
  for (const key of ['creationTimeMin', 'creationTimeMax', 'lastStatusOccurringTimeMin', 'lastStatusOccurringTimeMax'] as const) {
    if (future(f[key])) errors[key] = 'Cannot be in the future'
  }
  if (f.creationTimeMin && f.creationTimeMax && f.creationTimeMin > f.creationTimeMax) {
    errors.creationTimeMax = 'Must be after "from"'
  }
  if (f.lastStatusOccurringTimeMin && f.lastStatusOccurringTimeMax
    && f.lastStatusOccurringTimeMin > f.lastStatusOccurringTimeMax) {
    errors.lastStatusOccurringTimeMax = 'Must be after "from"'
  }
  return errors
}

export default function TrackingFilters({
  initial,
  onSearch,
  onClear,
}: {
  initial: Filters,
  onSearch: (filters: Filters) => void,
  onClear: () => void,
}) {
  const [form, setForm] = useState<FormState>(() => toForm(initial));
  const [errors, setErrors] = useState<ReturnType<typeof validate>>({});

  const field = (key: keyof FormState) => ({
    value: form[key],
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [key]: e.target.value }),
    error: errors[key] != null,
    helperText: errors[key],
  })

  const dateField = (key: keyof FormState, label: string) => (
    <TextField
      {...field(key)}
      label={label}
      type="datetime-local"
      size="small"
      slotProps={{ inputLabel: { shrink: true } }}
      sx={{ flex: 1, minWidth: '22ch' }}
    />
  )

  const handleSubmit = (e: React.SubmitEvent) => {
    e.preventDefault();
    const filters = toFilters(form);
    const found = validate(filters);
    setErrors(found);
    if (Object.keys(found).length === 0) onSearch(filters);
  }

  const handleClear = () => {
    setForm(toForm({}));
    setErrors({});
    onClear();
  }

  return (
    <Box component="form" onSubmit={handleSubmit} sx={{ margin: '1vh' }}>
      <Stack spacing={2}>
        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
          <TextField {...field('trackingNumber')} label="Tracking number contains" size="small" sx={{ flex: 2 }} />
          <TextField {...field('pol')} label="POL starts with" size="small" sx={{ flex: 1 }} />
          <TextField {...field('pod')} label="POD starts with" size="small" sx={{ flex: 1 }} />
        </Stack>
        <Box>
          <Typography variant="overline" color="text.secondary">Created</Typography>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            {dateField('creationTimeMin', 'From')}
            {dateField('creationTimeMax', 'To')}
          </Stack>
        </Box>
        <Box>
          <Typography variant="overline" color="text.secondary">Last status occurred</Typography>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            {dateField('lastStatusOccurringTimeMin', 'From')}
            {dateField('lastStatusOccurringTimeMax', 'To')}
          </Stack>
        </Box>
        <Stack direction="row" spacing={1} sx={{ justifyContent: 'flex-end' }}>
          <Button variant="text" onClick={handleClear}>Clear</Button>
          <Button variant="contained" type="submit" startIcon={<SearchIcon />}>Search</Button>
        </Stack>
      </Stack>
    </Box>
  )
}

export type { Filters }
