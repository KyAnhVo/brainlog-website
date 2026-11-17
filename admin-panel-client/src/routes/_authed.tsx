import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'
import { meQueryOptions } from '@/lib/auth';

// Pathless layout: every route under `_authed/` needs a valid session.
export const Route = createFileRoute('/_authed')({
  beforeLoad: async ({ context, location }) => {
    const me = await context.queryClient.ensureQueryData(meQueryOptions);
    if (!me) {
      throw redirect({
        to: '/login',
        search: { redirect: location.href },
        replace: true,
      })
    }
    return { me }
  },
  component: () => <Outlet />,
})
