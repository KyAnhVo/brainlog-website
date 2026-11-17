import { Alert, AlertTitle } from '@mui/material';
import { isApiError, type ApiError } from '@/lib/api';

function toApiError(error: unknown): ApiError {
  return isApiError(error) ? error
    : { errorCode: 0, errorMsg: error instanceof Error ? error.message : 'Network error' };
}

export default function ErrorAlert({ error }: { error: unknown }) {
  const err = toApiError(error);
  return (
    <Alert severity="error" variant="outlined">
      <AlertTitle>Error {err.errorCode}</AlertTitle>
      {err.errorMsg}
    </Alert>
  );
}
