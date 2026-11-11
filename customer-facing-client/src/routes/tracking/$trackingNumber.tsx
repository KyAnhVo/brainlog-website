import { Alert, AlertTitle, Box, Card, CardContent, Divider, List, ListItem, Typography } from '@mui/material';
import {
  Timeline,
  TimelineConnector,
  TimelineContent,
  TimelineDot,
  TimelineItem,
  TimelineOppositeContent,
  TimelineSeparator,
} from '@mui/lab';
import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query';
import { api, isApiError, type ApiError } from '@/lib/api';
import type { StatusCustomer, TrackingBase, TrackingResponse } from '@/lib/tracking';

export const Route = createFileRoute('/tracking/$trackingNumber')({
  component: RouteComponent,
})

async function fetchTracking(trackingNumber: string, signal: AbortSignal): Promise<TrackingResponse> {
  const { data } = await api.get<TrackingResponse>(`/tracking/${trackingNumber}`, { signal });
  return data;
}

function formatPhysicalDetails(p: NonNullable<TrackingBase['physicalDetails']>): string {
  const { lengthCm, widthCm, heightCm } = p.dimensions;
  const parts: string[] = [];
  if (p.pcsCount != null) parts.push(`${p.pcsCount} pcs`);
  parts.push(`${p.weightKg} kg`);
  parts.push(`${lengthCm} × ${widthCm} × ${heightCm} cm`);
  return parts.join(', ');
}

function dotColor(event: StatusCustomer): 'error' | 'warning' | 'primary' {
  if (event.statusType === 'bad') return 'error';
  return event.warnings.length > 0 ? 'warning' : 'primary';
}

function RouteComponent() {
  const { trackingNumber } = Route.useParams();
  const { data, error } = useQuery({
    queryKey: ['tracking', trackingNumber],
    queryFn: ({ signal }) => fetchTracking(encodeURIComponent(trackingNumber), signal),
    retry: false,
  });

  const statuses = data?.statuses ?? [];
  const details = data?.details ?? null;
  const physicalDetails = details?.physicalDetails ? formatPhysicalDetails(details.physicalDetails) : null;
  const err: ApiError | null = error == null ? null
    : isApiError(error) ? error
      : { errorCode: 0, errorMsg: error instanceof Error ? error.message : 'Network error' };

  return (
    <>
      <Divider variant="fullWidth" />
      <Box sx={{ margin: 2 }}>
        {
          err == null ?
            <>
              <Typography component="h1" variant="h4" gutterBottom sx={{ fontWeight: 700 }}>
                Tracking {trackingNumber}
              </Typography>
              <Card variant="outlined" sx={{ marginBottom: 3 }}>
                <CardContent>
                  <Typography component="h2" variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
                    Item Details
                  </Typography>
                  {details && (
                    <List dense>
                      <ListItem sx={{ gap: 1 }}>
                        <Typography color="text.secondary" sx={{ minWidth: 160, flexShrink: 0 }}><b>Tracking Number:</b></Typography>
                        <Typography>{details.trackingNumber}</Typography>
                      </ListItem>
                      {details.cont && (
                        <ListItem sx={{ gap: 1 }}>
                          <Typography color="text.secondary" sx={{ minWidth: 160, flexShrink: 0 }}><b>Container Number:</b></Typography>
                          <Typography>{details.cont}</Typography>
                        </ListItem>
                      )}
                      {details.seal && (
                        <ListItem sx={{ gap: 1 }}>
                          <Typography color="text.secondary" sx={{ minWidth: 160, flexShrink: 0 }}><b>Seal Number:</b></Typography>
                          <Typography>{details.seal}</Typography>
                        </ListItem>
                      )}
                      <ListItem sx={{ gap: 1 }}>
                        <Typography color="text.secondary" sx={{ minWidth: 160, flexShrink: 0 }}><b>Port of Loading:</b></Typography>
                        <Typography>{details.pol}</Typography>
                      </ListItem>
                      <ListItem sx={{ gap: 1 }}>
                        <Typography color="text.secondary" sx={{ minWidth: 160, flexShrink: 0 }}><b>Port of Discharge:</b></Typography>
                        <Typography>{details.pod}</Typography>
                      </ListItem>
                      {physicalDetails && (
                        <ListItem sx={{ gap: 1 }}>
                          <Typography color="text.secondary" sx={{ minWidth: 160, flexShrink: 0 }}><b>Physical Details:</b></Typography>
                          <Typography>{physicalDetails}</Typography>
                        </ListItem>
                      )}
                    </List>
                  )}
                </CardContent>
              </Card>
              <Timeline position="right" sx={{ padding: 0 }}>
                {statuses.map((event, i) => (
                  <TimelineItem key={event.id} sx={{ flexDirection: { xs: 'column', sm: 'row' } }}>
                    <TimelineOppositeContent
                      color="text.secondary"
                      sx={{ flex: 0.3, textAlign: { xs: 'left', sm: 'right' } }}
                    >
                      {new Date(event.occurringTime).toLocaleString()}
                    </TimelineOppositeContent>
                    <TimelineSeparator>
                      <TimelineDot color={dotColor(event)} />
                      {i < statuses.length - 1 && <TimelineConnector />}
                    </TimelineSeparator>
                    <TimelineContent>
                      {event.statusType === 'bad' ? (
                        <Typography color="error.main" sx={{ fontWeight: 600 }}>{event.status}</Typography>
                      ) : (
                        <Typography>{event.status}</Typography>
                      )}
                      <Typography variant="body2" color="text.secondary">
                        {event.location}
                      </Typography>
                      {event.statusType === 'good' && event.warnings.map((warning, wi) => (
                        <Typography key={wi} variant="body2" color="warning.main" sx={{ fontWeight: 600 }}>
                          {warning}
                        </Typography>
                      ))}
                    </TimelineContent>
                  </TimelineItem>
                ))}
              </Timeline>
            </>
            :
            <Alert severity="error" variant="outlined">
              <AlertTitle>Error {err.errorCode}</AlertTitle>
              {err.errorMsg}
            </Alert>
        }
      </Box>
    </>
  )
}
