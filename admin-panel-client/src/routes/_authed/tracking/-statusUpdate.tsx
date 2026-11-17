import {
  Alert, Box, Button, Card, CardContent, IconButton, MenuItem, Stack, TextField, ToggleButton,
  ToggleButtonGroup, Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import React, { useState } from 'react';
import { api, type ApiError } from '@/lib/api';
import { fromLocalInput, toLocalInput } from '@/lib/time';
import { GOOD_STATUS_NAMES, type GoodStatusName, type Status, type StatusCreate } from '@/lib/tracking';

async function postStatus(trackingNumber: string, body: StatusCreate): Promise<Status> {
  const { data } = await api.post<Status>(`/admin/tracking/${encodeURIComponent(trackingNumber)}/status`, body);
  return data;
}

// Suggest the step after the latest good status, so the common case is one click.
function suggestNext(statuses: Status[]): GoodStatusName {
  for (let i = statuses.length - 1; i >= 0; i--) {
    const s = statuses[i];
    if (s.statusType === 'good') {
      const idx = GOOD_STATUS_NAMES.indexOf(s.status);
      return GOOD_STATUS_NAMES[Math.min(idx + 1, GOOD_STATUS_NAMES.length - 1)];
    }
  }
  return GOOD_STATUS_NAMES[0];
}

type FormState = {
  statusType: 'good' | 'bad',
  goodStatus: GoodStatusName,
  badStatus: string,
  warnings: string[],
  location: string,
  occurringTime: string,  // datetime-local value
}

function initialForm(statuses: Status[]): FormState {
  return {
    statusType: 'good',
    goodStatus: suggestNext(statuses),
    badStatus: '',
    warnings: [],
    location: statuses[statuses.length - 1]?.location ?? '',
    occurringTime: toLocalInput(new Date().toISOString()),
  }
}

export default function StatusUpdate({ trackingNumber, statuses }: { trackingNumber: string, statuses: Status[] }) {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<FormState>(() => initialForm(statuses));
  const [timeError, setTimeError] = useState<string | null>(null);

  const mutation = useMutation<Status, ApiError, StatusCreate>({
    mutationFn: (body) => postStatus(trackingNumber, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-tracking', trackingNumber] });
      queryClient.invalidateQueries({ queryKey: ['admin-trackings'] });
    },
  });

  const set = <K extends keyof FormState>(key: K, value: FormState[K]) => setForm((f) => ({ ...f, [key]: value }));

  const handleSubmit = (e: React.SubmitEvent) => {
    e.preventDefault();
    const occurringTime = fromLocalInput(form.occurringTime);
    if (!occurringTime) return setTimeError('Required');
    if (Date.parse(occurringTime) > Date.now()) return setTimeError('Must not be in the future');
    setTimeError(null);

    const common = { occurringTime, location: form.location.trim() };
    const body: StatusCreate = form.statusType === 'good'
      ? { ...common, statusType: 'good', status: form.goodStatus, warnings: form.warnings.map((w) => w.trim()).filter((w) => w !== '') }
      : { ...common, statusType: 'bad', status: form.badStatus.trim() };

    mutation.mutate(body, {
      onSuccess: (created) => setForm({ ...initialForm([...statuses, created]), location: created.location }),
    });
  }

  return (
    <Card variant="outlined">
      <CardContent>
        <Typography component="h2" variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
          Add Status
        </Typography>
        <Box component="form" onSubmit={handleSubmit}>
          <Stack spacing={2}>
            <ToggleButtonGroup
              exclusive
              fullWidth
              size="small"
              value={form.statusType}
              onChange={(_, v: FormState['statusType'] | null) => v && set('statusType', v)}
            >
              <ToggleButton value="good" color="primary">Normal</ToggleButton>
              <ToggleButton value="bad" color="error">Problem</ToggleButton>
            </ToggleButtonGroup>

            {form.statusType === 'good' ? (
              <>
                <TextField
                  select
                  required
                  label="Status"
                  size="small"
                  value={form.goodStatus}
                  onChange={(e) => set('goodStatus', e.target.value as GoodStatusName)}
                >
                  {GOOD_STATUS_NAMES.map((name) => <MenuItem key={name} value={name}>{name}</MenuItem>)}
                </TextField>
                <Box>
                  <Typography variant="overline" color="text.secondary">Warnings</Typography>
                  <Stack spacing={1}>
                    {form.warnings.map((w, i) => (
                      <Stack direction="row" spacing={1} key={i}>
                        <TextField
                          size="small"
                          fullWidth
                          placeholder="e.g. Delayed by customs inspection"
                          value={w}
                          onChange={(e) => set('warnings', form.warnings.map((x, j) => (j === i ? e.target.value : x)))}
                        />
                        <IconButton
                          aria-label="Remove warning"
                          onClick={() => set('warnings', form.warnings.filter((_, j) => j !== i))}
                        >
                          <DeleteOutlinedIcon />
                        </IconButton>
                      </Stack>
                    ))}
                    <Button
                      size="small"
                      color="warning"
                      startIcon={<AddIcon />}
                      onClick={() => set('warnings', [...form.warnings, ''])}
                      sx={{ alignSelf: 'flex-start' }}
                    >
                      Add warning
                    </Button>
                  </Stack>
                </Box>
              </>
            ) : (
              <TextField
                required
                label="Problem description"
                placeholder="e.g. Rejected by customs: prohibited item"
                size="small"
                multiline
                minRows={2}
                value={form.badStatus}
                onChange={(e) => set('badStatus', e.target.value)}
              />
            )}

            <TextField
              required
              label="Location"
              size="small"
              value={form.location}
              onChange={(e) => set('location', e.target.value)}
            />
            <TextField
              required
              label="Occurred at"
              type="datetime-local"
              size="small"
              value={form.occurringTime}
              onChange={(e) => set('occurringTime', e.target.value)}
              error={timeError != null}
              helperText={timeError ?? 'Your local time'}
              slotProps={{ inputLabel: { shrink: true } }}
            />

            {mutation.isError && (
              <Alert severity="error" variant="outlined">
                Error {mutation.error.errorCode}: {mutation.error.errorMsg}
              </Alert>
            )}
            {mutation.isSuccess && (
              <Alert severity="success" variant="outlined" onClose={() => mutation.reset()}>
                Status “{mutation.data.status}” added.
              </Alert>
            )}

            <Button
              variant="contained"
              type="submit"
              color={form.statusType === 'bad' ? 'error' : 'primary'}
              disabled={mutation.isPending}
            >
              {mutation.isPending ? 'Saving…' : 'Add status'}
            </Button>
          </Stack>
        </Box>
      </CardContent>
    </Card>
  )
}
