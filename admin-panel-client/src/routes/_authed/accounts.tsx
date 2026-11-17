import {
  Alert, Box, Button, Card, CardContent, CardHeader, Stack, TextField, ToggleButton, ToggleButtonGroup, Typography,
} from '@mui/material';
import PersonAddIcon from '@mui/icons-material/PersonAdd';
import PersonRemoveIcon from '@mui/icons-material/PersonRemove';
import { createFileRoute, redirect, useNavigate } from '@tanstack/react-router'
import { useMutation, useQueryClient } from '@tanstack/react-query';
import React, { useState } from 'react';

import logo from '@/assets/logo.png';
import { api, type ApiError } from '@/lib/api';
import type { AccountCreate, AccountDelete, AccountPublic, Role } from '@/lib/account';
import { meQueryKey } from '@/lib/auth';

export const Route = createFileRoute('/_authed/accounts')({
  // Only admins may manage accounts; everyone else is sent back to the tracking list.
  beforeLoad: ({ context }) => {
    if (context.me.role !== 'admin') throw redirect({ to: '/tracking', replace: true })
  },
  component: RouteComponent,
})

const MIN_PASSWORD_LENGTH = 8

function RouteComponent() {
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
        maxWidth: 1100,
      }}>
        <CardHeader
          title={
            <Typography
              component="h1"
              sx={{
                fontSize: "40px",
              }}
            >
              Accounts
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
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={3} sx={{ alignItems: 'flex-start' }}>
            <CreateAccount />
            <DeleteAccount />
          </Stack>
        </CardContent>
      </Card>
    </Box>
  )
}

function createErrorMessage(e: ApiError): string {
  switch (e.errorCode) {
    case 409: return 'An account with this email already exists.';
    case 401: return 'You are not allowed to create accounts.';
    case 500: return 'Server error: the account was not created. Please try again.';
    default: return `Error ${e.errorCode}: ${e.errorMsg}`;
  }
}

function CreateAccount() {
  const [form, setForm] = useState<AccountCreate>({ email: '', password: '', name: '', role: 'user' });
  const [confirm, setConfirm] = useState('');

  const mutation = useMutation<AccountPublic, ApiError, AccountCreate>({
    mutationFn: async (body) => (await api.post<AccountPublic>('/admin/createAccount', body)).data,
    onSuccess: () => {
      setForm({ email: '', password: '', name: '', role: 'user' });
      setConfirm('');
    },
  });

  const passwordTooShort = form.password.length > 0 && form.password.length < MIN_PASSWORD_LENGTH;
  const mismatch = confirm.length > 0 && confirm !== form.password;

  const handleSubmit = (e: React.SubmitEvent) => {
    e.preventDefault();
    if (form.password.length < MIN_PASSWORD_LENGTH || confirm !== form.password) return;
    mutation.mutate({ ...form, name: form.name.trim(), email: form.email.trim() });
  }

  return (
    <Card variant="outlined" sx={{ flex: 1, width: '100%' }}>
      <CardContent>
        <Typography component="h2" variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
          Create Account
        </Typography>
        <Box component="form" onSubmit={handleSubmit}>
          <Stack spacing={2}>
            <TextField
              required size="small" label="Name"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
            />
            <TextField
              required size="small" label="Email" type="email" autoComplete="off"
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
            />
            <TextField
              required size="small" label="Password" type="password" autoComplete="new-password"
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
              error={passwordTooShort}
              helperText={`At least ${MIN_PASSWORD_LENGTH} characters`}
            />
            <TextField
              required size="small" label="Confirm password" type="password" autoComplete="new-password"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              error={mismatch}
              helperText={mismatch ? 'Passwords do not match' : ' '}
            />
            <Box>
              <Typography variant="overline" color="text.secondary">Role</Typography>
              <ToggleButtonGroup
                exclusive fullWidth size="small"
                value={form.role}
                onChange={(_, v: Role | null) => v && setForm({ ...form, role: v })}
              >
                <ToggleButton value="user">User</ToggleButton>
                <ToggleButton value="admin">Admin</ToggleButton>
              </ToggleButtonGroup>
            </Box>
            {mutation.isError && <Alert severity="error" variant="outlined">{createErrorMessage(mutation.error)}</Alert>}
            {mutation.isSuccess && (
              <Alert severity="success" variant="outlined" onClose={() => mutation.reset()}>
                Account “{mutation.data.name}” created. ID: <code>{mutation.data.id}</code>
              </Alert>
            )}
            <Button variant="contained" type="submit" startIcon={<PersonAddIcon />} disabled={mutation.isPending}>
              {mutation.isPending ? 'Creating…' : 'Create account'}
            </Button>
          </Stack>
        </Box>
      </CardContent>
    </Card>
  )
}

function deleteErrorMessage(e: ApiError): string {
  switch (e.errorCode) {
    case 404: return 'No account found.';
    case 401: return 'You are not allowed to delete accounts.';
    default: return `Error ${e.errorCode}: ${e.errorMsg}`;
  }
}

function DeleteAccount() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [by, setBy] = useState<'email' | 'id'>('email');
  const [value, setValue] = useState('');
  const [confirming, setConfirming] = useState(false);

  const mutation = useMutation<unknown, ApiError, AccountDelete>({
    mutationFn: (body) => api.post('/admin/deleteAccount', body),
    onSuccess: async () => {
      setValue('');
      queryClient.invalidateQueries({ queryKey: ['account'] });  // name lookups may now 404
      // Deleting your own account ends your session: re-check it and leave if it is gone.
      await queryClient.invalidateQueries({ queryKey: meQueryKey });
      if (!queryClient.getQueryData(meQueryKey)) navigate({ to: '/login', replace: true });
    },
    onSettled: () => setConfirming(false),
  });

  const handleSubmit = (e: React.SubmitEvent) => {
    e.preventDefault();
    if (!confirming) return setConfirming(true);
    const v = value.trim();
    mutation.mutate(by === 'email' ? { email: v } : { id: v });
  }

  return (
    <Card variant="outlined" sx={{ flex: 1, width: '100%' }}>
      <CardContent>
        <Typography component="h2" variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
          Delete Account
        </Typography>
        <Box component="form" onSubmit={handleSubmit}>
          <Stack spacing={2}>
            <ToggleButtonGroup
              exclusive fullWidth size="small"
              value={by}
              onChange={(_, v: 'email' | 'id' | null) => { if (v) { setBy(v); setConfirming(false); } }}
            >
              <ToggleButton value="email">By email</ToggleButton>
              <ToggleButton value="id">By account ID</ToggleButton>
            </ToggleButtonGroup>
            <TextField
              required size="small"
              label={by === 'email' ? 'Email' : 'Account ID'}
              type={by === 'email' ? 'email' : 'text'}
              autoComplete="off"
              value={value}
              onChange={(e) => { setValue(e.target.value); setConfirming(false); }}
            />
            {confirming && (
              <Alert severity="warning" variant="outlined">
                This permanently deletes <b>{value.trim()}</b> and signs it out everywhere. Press again to confirm.
              </Alert>
            )}
            {mutation.isError && <Alert severity="error" variant="outlined">{deleteErrorMessage(mutation.error)}</Alert>}
            {mutation.isSuccess && (
              <Alert severity="success" variant="outlined" onClose={() => mutation.reset()}>
                Account deleted.
              </Alert>
            )}
            <Stack direction="row" spacing={1}>
              {confirming && <Button onClick={() => setConfirming(false)}>Cancel</Button>}
              <Button
                variant={confirming ? 'contained' : 'outlined'}
                color="error"
                type="submit"
                startIcon={<PersonRemoveIcon />}
                disabled={mutation.isPending}
                sx={{ flex: 1 }}
              >
                {mutation.isPending ? 'Deleting…' : confirming ? 'Confirm delete' : 'Delete account'}
              </Button>
            </Stack>
          </Stack>
        </Box>
      </CardContent>
    </Card>
  )
}
