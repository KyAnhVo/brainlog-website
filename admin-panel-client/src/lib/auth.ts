import { queryOptions } from '@tanstack/react-query'
import { api, isApiError } from '@/lib/api'
import type { LoginSuccessResponse } from '@/lib/account'

// The session cookie is HttpOnly, so the client cannot inspect it. Instead it asks
// the server who it is. ASSUMED (unspecified): GET /api/admin/me -> 200 LoginSuccessResponse | 401.
// A 401 resolves to `null` (logged out) instead of an error.
async function fetchMe(signal: AbortSignal): Promise<LoginSuccessResponse | null> {
  try {
    const { data } = await api.get<LoginSuccessResponse>('/admin/me', { signal })
    return data
  } catch (e) {
    if (isApiError(e) && e.errorCode === 401) return null
    throw e
  }
}

const meQueryKey = ['me'] as const

const meQueryOptions = queryOptions({
  queryKey: meQueryKey,
  queryFn: ({ signal }) => fetchMe(signal),
  staleTime: 5 * 60_000,
  retry: false,
})

export { meQueryKey, meQueryOptions }
