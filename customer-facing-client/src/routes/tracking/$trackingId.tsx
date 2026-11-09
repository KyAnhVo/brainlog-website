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

export const Route = createFileRoute('/tracking/$trackingId')({
  component: RouteComponent,
})

/**
 * A good status event, i.e. normal progress of the shipment:
 * - status (string): the status reached (e.g. customs clearance, arrived at POL, etc.)
 * - occurringTime (Date): when the status was achieved
 * - location (string): where that action happens
 * - warnings (string[]): non-fatal issues (customs adding money, delays, etc.)
 * Creation time and status creator are admin panel only, so they are not sent here.
 */
interface StatusEvent {
  type: 'good',
  status: string,
  occurringTime: Date,
  location: string,
  warnings: string[],
}

/**
 * A bad status event, i.e. something went wrong with the cargo
 * (fell off a ship, rejected by customs, etc.):
 * - status (string): what happened
 * - occurringTime, location: same as StatusEvent
 */
interface StatusErrorEvent {
  type: 'bad',
  status: string,
  occurringTime: Date,
  location: string,
}

type TrackingEvent = StatusEvent | StatusErrorEvent;

/** Dates arrive over the wire as ISO-8601 strings. */
type Wire<T> = { [K in keyof T]: T[K] extends Date ? string : T[K] };

/**
 * This is the details of a shipment. There are:
 * - Tracking number
 * - Cont/Seal number
 * - POL (Port of Loading)
 * - POD (Port of Discharge)
 * - Physical details: pieces count, weight (kg), dimensions (m); each optional
 */
interface Details {
  trackingNumber: string,
  contOrSeal?: string,
  pol: string,
  pod: string,
  physicalDetails?: {
    pieces?: number,
    weightKg?: number,
    dimensionsM?: {
      length: number,
      width: number,
      height: number,
    },
  },
}

interface TrackingResponse {
  details: Details,
  statuses: (Wire<StatusEvent> | Wire<StatusErrorEvent>)[],
}

async function fetchTracking(trackingId: string, signal: AbortSignal): Promise<{ statuses: TrackingEvent[], details: Details }> {
  const { data } = await api.get<TrackingResponse>(`/tracking/${trackingId}`, { signal });

  const { statuses, details } = data;
  return {
    statuses: statuses.map((status) => ({
      ...status,
      occurringTime: new Date(status.occurringTime),
    }) as TrackingEvent),
    details,
  };
}

function formatPhysicalDetails(p: NonNullable<Details['physicalDetails']>): string | null {
  const parts: string[] = [];
  if (p.pieces != null) parts.push(`${p.pieces} pcs`);
  if (p.weightKg != null) parts.push(`${p.weightKg} kg`);
  if (p.dimensionsM != null) {
    const { length, width, height } = p.dimensionsM;
    parts.push(`${length} × ${width} × ${height} m`);
  }
  return parts.length > 0 ? parts.join(', ') : null;
}

function dotColor(event: TrackingEvent): 'error' | 'warning' | 'primary' {
  if (event.type === 'bad') return 'error';
  return event.warnings.length > 0 ? 'warning' : 'primary';
}

function RouteComponent() {
  const { trackingId } = Route.useParams();
  const { data, error } = useQuery({
    queryKey: ['tracking', trackingId],
    queryFn: ({ signal }) => fetchTracking(encodeURIComponent(trackingId), signal),
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
                Tracking {trackingId}
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
                      {details.contOrSeal && (
                        <ListItem sx={{ gap: 1 }}>
                          <Typography color="text.secondary" sx={{ minWidth: 160, flexShrink: 0 }}><b>Cont/Seal Number:</b></Typography>
                          <Typography>{details.contOrSeal}</Typography>
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
                  <TimelineItem key={i} sx={{ flexDirection: { xs: 'column', sm: 'row' } }}>
                    <TimelineOppositeContent
                      color="text.secondary"
                      sx={{ flex: 0.3, textAlign: { xs: 'left', sm: 'right' } }}
                    >
                      {event.occurringTime.toLocaleString()}
                    </TimelineOppositeContent>
                    <TimelineSeparator>
                      <TimelineDot color={dotColor(event)} />
                      {i < statuses.length - 1 && <TimelineConnector />}
                    </TimelineSeparator>
                    <TimelineContent>
                      {event.type === 'bad' ? (
                        <Typography color="error.main" sx={{ fontWeight: 600 }}>{event.status}</Typography>
                      ) : (
                        <Typography>{event.status}</Typography>
                      )}
                      <Typography variant="body2" color="text.secondary">
                        {event.location}
                      </Typography>
                      {event.type === 'good' && event.warnings.map((warning, wi) => (
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
