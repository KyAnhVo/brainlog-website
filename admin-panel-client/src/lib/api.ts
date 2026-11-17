import axios, { isAxiosError } from 'axios'

// API root, e.g. "/api" (same origin) or "https://api.example.com/api". No trailing slash.
const base = (import.meta.env.VITE_API_BASE_URL || '/api').replace(/\/+$/, '')

interface ApiError {
  errorCode: number,
  errorMsg: string,
}

function isApiError(e: unknown): e is ApiError {
  return typeof e === 'object' && e !== null && 'errorCode' in e && 'errorMsg' in e;
}

const api = axios.create({
  baseURL: base,
  timeout: 10_000,
  // The session lives in a cookie; send it even if the API is on another origin.
  withCredentials: true,
})

// Normalize every failure into an ApiError so callers never see AxiosError, SyntaxError, etc.
api.interceptors.response.use(
  (res) => {
    // A 2xx that isn't JSON (e.g. a SPA fallback serving index.html for /api) is a misrouted request.
    const contentType = String(res.headers['content-type'] ?? '')
    if (!contentType.includes('application/json') || typeof res.data !== 'object' || res.data === null) {
      const err: ApiError = { errorCode: res.status, errorMsg: 'Unexpected response from server' }
      return Promise.reject(err)
    }
    return res
  },
  (error: unknown) => Promise.reject(toApiError(error)),
)

function toApiError(error: unknown): ApiError {
  if (!isAxiosError(error)) {
    return { errorCode: 0, errorMsg: error instanceof Error ? error.message : 'Unknown error' }
  }
  if (error.response) {
    const { status, statusText, data } = error.response
    return isApiError(data) ? data : { errorCode: status, errorMsg: statusText || 'Request failed' }
  }
  if (error.code === 'ECONNABORTED' || error.code === 'ETIMEDOUT') {
    return { errorCode: 0, errorMsg: 'Request timed out' }
  }
  return { errorCode: 0, errorMsg: 'Unable to reach server' }
}

export { api, base, isApiError }
export type { ApiError }
