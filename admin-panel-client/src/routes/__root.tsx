import * as React from 'react'
import { Link, Outlet, createRootRouteWithContext, useNavigate } from '@tanstack/react-router'
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools'
import { useMutation, useQuery, type QueryClient } from '@tanstack/react-query'
import { AppBar, Box, Button, Chip, Toolbar, Typography } from '@mui/material';
import LogoutIcon from '@mui/icons-material/Logout';

import logo from '@/assets/logo.png';
import { api } from '@/lib/api';
import { meQueryKey, meQueryOptions } from '@/lib/auth';

type RouterContext = {
  queryClient: QueryClient,
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: RootComponent,
})

function RootComponent() {
  return (
    <React.Fragment>
      <NavBar />
      <Outlet />
      <TanStackRouterDevtools />
    </React.Fragment>
  )
}

function NavBar() {
  const { queryClient } = Route.useRouteContext();
  const navigate = useNavigate();
  const { data: me } = useQuery(meQueryOptions);

  const logout = useMutation({
    mutationFn: () => api.post('/admin/logout'),
    // Even if the server call fails, drop the local session state.
    onSettled: () => {
      queryClient.clear();
      queryClient.setQueryData(meQueryKey, null);
      navigate({ to: '/login', replace: true });
    },
  });

  const pages = [
    { name: "Tracking", to: "/tracking" },
    ...(me?.role === 'admin' ? [{ name: "Accounts", to: "/accounts" }] : []),
  ] as const

  return (
    <AppBar position="sticky" sx={{ minWidth: '100vh' }}>
      <Toolbar>
        {/* Branding: logo + name, sits on the left */}
        <Box
          component="img"
          src={logo}          // swap for your actual logo path
          alt="logo"
          sx={{ height: 32, mr: 1.5 }}
        />

        <Typography
          variant="h6"
          component="div"
          sx={{
            fontSize: "2vh"
          }}
        >
          BrainLog Tracking Admin Panel
        </Typography>

        {/* The spacer: eats leftover space, pushing the menu to the right */}
        <Box sx={{ flexGrow: 1 }} />

        {me && (
          <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
            {/* Menu items, generated from the array */}
            {pages.map((page) => (
              <Button
                component={Link}
                to={page.to}
                key={page.name}
                color="inherit"
                activeProps={{ style: { textDecoration: 'underline', textUnderlineOffset: 6 } }}
              >
                {page.name}
              </Button>
            ))}
            <Typography variant="body2" sx={{ ml: 2 }}>{me.name}</Typography>
            <Chip
              label={me.role}
              size="small"
              variant="outlined"
              sx={{ color: 'inherit', borderColor: 'rgba(255,255,255,0.6)' }}
            />
            <Button
              color="inherit"
              startIcon={<LogoutIcon />}
              onClick={() => logout.mutate()}
              disabled={logout.isPending}
            >
              Log out
            </Button>
          </Box>
        )}
      </Toolbar>
    </AppBar>
  );
}
