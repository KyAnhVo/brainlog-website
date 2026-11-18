import {
  Alert, Box, Button, Card, CardContent, CardHeader, Collapse, FormControlLabel, InputAdornment, Stack, Switch,
  TextField, Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { useMutation, useQueryClient } from '@tanstack/react-query';
import React, { useState } from 'react';

import logo from '@/assets/logo.png';
import { api, type ApiError } from '@/lib/api';
import type { AdminTrackingResponse, TrackingCreate } from '@/lib/tracking';

export const Route = createFileRoute('/_authed/tracking/new')({
  component: RouteComponent,
})

async function postTracking(body: TrackingCreate): Promise<AdminTrackingResponse> {
  const { data } = await api.post<AdminTrackingResponse>('/admin/tracking', body);
  return data;
}

type FormState = {
  pol: string,
  pod: string,
  cont: string,
  seal: string,
  withPhysical: boolean,
  pcsCount: string,
  weightKg: string,
  lengthCm: string,
  widthCm: string,
  heightCm: string,
}

const EMPTY: FormState = {
  pol: '', pod: '', cont: '', seal: '',
  withPhysical: false, pcsCount: '', weightKg: '', lengthCm: '', widthCm: '', heightCm: '',
}

type NumberField = 'pcsCount' | 'weightKg' | 'lengthCm' | 'widthCm' | 'heightCm'

// Mirrors the server's rules so mistakes show next to the field before a round trip.
function validate(f: FormState): Partial<Record<NumberField, string>> {
  const errors: Partial<Record<NumberField, string>> = {}
  if (!f.withPhysical) return errors
  for (const key of ['weightKg', 'lengthCm', 'widthCm', 'heightCm'] as const) {
    const n = Number(f[key])
    if (f[key].trim() === '' || !Number.isFinite(n) || n <= 0) errors[key] = 'Must be greater than 0'
  }
  if (f.pcsCount.trim() !== '') {
    const n = Number(f.pcsCount)
    if (!Number.isInteger(n) || n < 1) errors.pcsCount = 'Must be a whole number, at least 1'
  }
  return errors
}

function toRequest(f: FormState): TrackingCreate {
  const body: TrackingCreate = { pol: f.pol.trim(), pod: f.pod.trim() }
  if (f.cont.trim()) body.cont = f.cont.trim()
  if (f.seal.trim()) body.seal = f.seal.trim()
  if (f.withPhysical) {
    body.physicalDetails = {
      weightKg: Number(f.weightKg),
      dimensions: { lengthCm: Number(f.lengthCm), widthCm: Number(f.widthCm), heightCm: Number(f.heightCm) },
      ...(f.pcsCount.trim() !== '' ? { pcsCount: Number(f.pcsCount) } : {}),
    }
  }
  return body
}

function RouteComponent() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [form, setForm] = useState<FormState>(EMPTY);
  const [errors, setErrors] = useState<ReturnType<typeof validate>>({});

  const mutation = useMutation<AdminTrackingResponse, ApiError, TrackingCreate>({
    mutationFn: postTracking,
    onSuccess: (created) => {
      const trackingNumber = created.details.trackingNumber;
      queryClient.setQueryData(['admin-tracking', trackingNumber], created);
      queryClient.invalidateQueries({ queryKey: ['admin-trackings'] });
      navigate({ to: '/tracking/$trackingNumber', params: { trackingNumber }, replace: true });
    },
  });

  const text = (key: 'pol' | 'pod' | 'cont' | 'seal') => ({
    value: form[key],
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [key]: e.target.value }),
  })

  const number = (key: NumberField, label: string, unit?: string, required = true) => (
    <TextField
      label={label}
      required={required}
      size="small"
      type="number"
      value={form[key]}
      onChange={(e) => setForm({ ...form, [key]: e.target.value })}
      error={errors[key] != null}
      helperText={errors[key]}
      slotProps={{
        htmlInput: { min: 0, step: key === 'pcsCount' ? 1 : 'any' },
        input: unit ? { endAdornment: <InputAdornment position="end">{unit}</InputAdornment> } : undefined,
      }}
      sx={{ flex: 1, minWidth: '16ch' }}
    />
  )

  const handleSubmit = (e: React.SubmitEvent) => {
    e.preventDefault();
    const found = validate(form);
    setErrors(found);
    if (Object.keys(found).length === 0) mutation.mutate(toRequest(form));
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
        maxWidth: 800,
      }}>
        <CardHeader
          title={
            <Typography
              component="h1"
              sx={{
                fontSize: "40px",
              }}
            >
              New Tracking
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
          <Box component="form" onSubmit={handleSubmit}>
            <Stack spacing={3}>
              <Alert severity="info" variant="outlined">
                The tracking number is generated automatically. A “Tracking created” status is added at the port of
                loading.
              </Alert>

              <Box>
                <Typography variant="overline" color="text.secondary">Route</Typography>
                <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
                  <TextField {...text('pol')} required size="small" label="Port of Loading (POL)" placeholder="e.g. Hải Phòng, VN" sx={{ flex: 1 }} />
                  <TextField {...text('pod')} required size="small" label="Port of Discharge (POD)" placeholder="e.g. Singapore, SG" sx={{ flex: 1 }} />
                </Stack>
              </Box>

              <Box>
                <Typography variant="overline" color="text.secondary">Container (optional)</Typography>
                <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
                  <TextField {...text('cont')} size="small" label="Container Number" sx={{ flex: 1 }} />
                  <TextField {...text('seal')} size="small" label="Seal Number" sx={{ flex: 1 }} />
                </Stack>
              </Box>

              <Box>
                <FormControlLabel
                  control={
                    <Switch
                      checked={form.withPhysical}
                      onChange={(e) => { setForm({ ...form, withPhysical: e.target.checked }); setErrors({}); }}
                    />
                  }
                  label="Add physical details"
                />
                <Collapse in={form.withPhysical} unmountOnExit>
                  <Stack spacing={2} sx={{ mt: 1 }}>
                    <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
                      {number('weightKg', 'Weight', 'kg')}
                      {number('pcsCount', 'Pieces (optional)', 'pcs', false)}
                    </Stack>
                    <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
                      {number('lengthCm', 'Length', 'cm')}
                      {number('widthCm', 'Width', 'cm')}
                      {number('heightCm', 'Height', 'cm')}
                    </Stack>
                  </Stack>
                </Collapse>
              </Box>

              {mutation.isError && (
                <Alert severity="error" variant="outlined">
                  Error {mutation.error.errorCode}: {mutation.error.errorMsg}
                </Alert>
              )}

              <Stack direction="row" spacing={1} sx={{ justifyContent: 'flex-end' }}>
                <Button component={Link} to="/tracking">Cancel</Button>
                <Button variant="contained" type="submit" startIcon={<AddIcon />} disabled={mutation.isPending}>
                  {mutation.isPending ? 'Creating…' : 'Create tracking'}
                </Button>
              </Stack>
            </Stack>
          </Box>
        </CardContent>
      </Card>
    </Box>
  )
}
