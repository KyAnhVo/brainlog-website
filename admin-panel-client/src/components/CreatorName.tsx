import { Tooltip, Typography, type TypographyProps } from '@mui/material';
import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';
import type { AccountPublic } from '@/lib/account';

async function fetchAccount(id: string, signal: AbortSignal): Promise<AccountPublic> {
  const { data } = await api.get<AccountPublic>(`/admin/account/${encodeURIComponent(id)}`, { signal });
  return data;
}

// Resolves a creatorId to a display name; falls back to the raw id (e.g. deleted account).
export default function CreatorName({ id, ...props }: { id: string } & TypographyProps) {
  const { data, isPending, isError } = useQuery({
    queryKey: ['account', id],
    queryFn: ({ signal }) => fetchAccount(id, signal),
    staleTime: Infinity,
    retry: false,
  });

  const label = isPending ? '…' : isError ? 'Unknown account' : data.name;

  return (
    <Tooltip title={`Account ID: ${id}`}>
      <Typography component="span" {...props} sx={{ fontStyle: isError ? 'italic' : undefined, ...props.sx }}>
        {label}
      </Typography>
    </Tooltip>
  );
}
