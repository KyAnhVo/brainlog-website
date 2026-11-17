import { Alert, Box, Button, Card, CardContent, CardHeader, TextField, Typography } from '@mui/material';
import { createFileRoute, redirect, useRouter } from '@tanstack/react-router'
import { useMutation } from '@tanstack/react-query';
import React, { useState } from 'react';

import logo from '@/assets/logo.png';
import { api, type ApiError } from '@/lib/api';
import type { Login, LoginSuccessResponse } from '@/lib/account';
import { meQueryKey, meQueryOptions } from '@/lib/auth';
import { safeRedirect } from '@/lib/redirect';

type LoginSearch = {
  redirect?: string,
}

export const Route = createFileRoute('/login')({
  validateSearch: (search: Record<string, unknown>): LoginSearch => ({
    redirect: typeof search.redirect === 'string' ? search.redirect : undefined,
  }),
  // Already logged in: skip the form.
  beforeLoad: async ({ context, search }) => {
    const me = await context.queryClient.ensureQueryData(meQueryOptions);
    if (me) throw redirect({ href: safeRedirect(search.redirect), replace: true })
  },
  component: RouteComponent,
})

async function postLogin(body: Login): Promise<LoginSuccessResponse> {
  const { data } = await api.post<LoginSuccessResponse>('/admin/login', body);
  return data;
}

function RouteComponent() {
  const { queryClient } = Route.useRouteContext();
  const search = Route.useSearch();
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  const login = useMutation<LoginSuccessResponse, ApiError, Login>({
    mutationFn: postLogin,
    onSuccess: (me) => {
      queryClient.setQueryData(meQueryKey, me);
      router.navigate({ href: safeRedirect(search.redirect), replace: true });
    },
  });

  const handleSubmit = (e: React.SubmitEvent) => {
    e.preventDefault();
    login.mutate({ email, password });
  }

  const errorMsg = login.error == null ? null
    : login.error.errorCode === 401 ? 'Invalid email or password.'
      : `Error ${login.error.errorCode}: ${login.error.errorMsg}`;

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
        minWidth: '25%'
      }}>
        <CardHeader
          title={
            <Typography
              component="h1"
              sx={{
                fontSize: "40px",
              }}
            >
              Admin Login
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
          <Box
            component="form" onSubmit={handleSubmit}
            sx={{
              display: 'flex',
              flexDirection: 'column',
              gap: 2,
              margin: '1vh',
              minWidth: '40ch',
            }}
          >
            <TextField
              required
              id="login-email-field"
              label="Email"
              type="email"
              autoComplete="username"
              autoFocus
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
            <TextField
              required
              id="login-password-field"
              label="Password"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
            {errorMsg && <Alert severity="error" variant="outlined">{errorMsg}</Alert>}
            <Button variant='contained' type='submit' disabled={login.isPending}>
              {login.isPending ? 'Logging in…' : 'Log in'}
            </Button>
          </Box>
        </CardContent>
      </Card>
    </Box>
  )
}
