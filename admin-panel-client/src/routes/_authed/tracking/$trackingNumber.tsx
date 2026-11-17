import { Box, Button, Card, CardContent, CircularProgress, List, ListItem, Stack, Typography } from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import {
  Timeline,
  TimelineConnector,
  TimelineContent,
  TimelineDot,
  TimelineItem,
  TimelineOppositeContent,
  TimelineSeparator,
} from '@mui/lab';
import { createFileRoute, Link, useCanGoBack, useRouter } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query';
import React from 'react';

import CreatorName from '@/components/CreatorName';
import ErrorAlert from '@/components/ErrorAlert';
import StatusChip from '@/components/StatusChip';
import { api } from '@/lib/api';
import { formatTime } from '@/lib/time';
import type { AdminTrackingResponse, StatusCustomer, TrackingBase } from '@/lib/tracking';
import StatusUpdate from './-statusUpdate.tsx'

export const Route = createFileRoute('/_authed/tracking/$trackingNumber')({
  component: RouteComponent,
})

async function fetchTracking(trackingNumber: string, signal: AbortSignal): Promise<AdminTrackingResponse> {
  const { data } = await api.get<AdminTrackingResponse>(`/admin/tracking/${encodeURIComponent(trackingNumber)}`, { signal });
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

function DetailRow({ label, children }: { label: string, children: React.ReactNode }) {
  return (
    <ListItem sx={{ gap: 1 }}>
      <Typography color="text.secondary" sx={{ minWidth: 160, flexShrink: 0 }}><b>{label}:</b></Typography>
      <Typography component="div">{children}</Typography>
    </ListItem>
  );
}

function BackButton() {
  const router = useRouter();
  const canGoBack = useCanGoBack();
  // Going back keeps the list's filters/sort/page; a direct visit falls back to the list page.
  return canGoBack
    ? <Button startIcon={<ArrowBackIcon />} onClick={() => router.history.back()}>Back to list</Button>
    : <Button startIcon={<ArrowBackIcon />} component={Link} to="/tracking">Back to list</Button>;
}

function RouteComponent() {
  const { trackingNumber } = Route.useParams();
  const { data, error, isPending } = useQuery({
    queryKey: ['admin-tracking', trackingNumber],
    queryFn: ({ signal }) => fetchTracking(trackingNumber, signal),
    retry: false,
  });

  const statuses = data?.statuses ?? [];
  const details = data?.details ?? null;
  const lastStatus = statuses[statuses.length - 1];

  return (
    <Box sx={{
      minHeight: '100vh',
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'flex-start'
    }}>
      <Box sx={{ margin: '5vh', width: '100%', maxWidth: 1400 }}>
        <BackButton />
        <Typography component="h1" variant="h4" gutterBottom sx={{ fontWeight: 700, mt: 1 }}>
          Tracking {trackingNumber}
        </Typography>
        {
          error != null ? <ErrorAlert error={error} />
            : isPending ? <CircularProgress />
              : details && (
                <Stack direction={{ xs: 'column', md: 'row' }} spacing={3} sx={{ alignItems: 'flex-start' }}>
                  <Stack spacing={3} sx={{ flex: 3, minWidth: 0, width: '100%' }}>
                    <Card variant="outlined">
                      <CardContent>
                        <Typography component="h2" variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
                          Item Details
                        </Typography>
                        <List dense>
                          <DetailRow label="Tracking Number">{details.trackingNumber}</DetailRow>
                          {details.cont && <DetailRow label="Container Number">{details.cont}</DetailRow>}
                          {details.seal && <DetailRow label="Seal Number">{details.seal}</DetailRow>}
                          <DetailRow label="Port of Loading">{details.pol}</DetailRow>
                          <DetailRow label="Port of Discharge">{details.pod}</DetailRow>
                          {details.physicalDetails && (
                            <DetailRow label="Physical Details">{formatPhysicalDetails(details.physicalDetails)}</DetailRow>
                          )}
                          {lastStatus && <DetailRow label="Last Status"><StatusChip status={lastStatus} /></DetailRow>}
                          <DetailRow label="Customer ID">{details.customerId ?? '—'}</DetailRow>
                          <DetailRow label="Created By"><CreatorName id={details.creatorId} /></DetailRow>
                          <DetailRow label="Created At">{formatTime(details.creationTime)}</DetailRow>
                        </List>
                      </CardContent>
                    </Card>

                    <Card variant="outlined">
                      <CardContent>
                        <Typography component="h2" variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
                          Status History ({statuses.length})
                        </Typography>
                        <Timeline position="right" sx={{ padding: 0 }}>
                          {statuses.map((event, i) => (
                            <TimelineItem key={event.id} sx={{ flexDirection: { xs: 'column', sm: 'row' } }}>
                              <TimelineOppositeContent
                                color="text.secondary"
                                sx={{ flex: 0.3, textAlign: { xs: 'left', sm: 'right' } }}
                              >
                                {formatTime(event.occurringTime)}
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
                                <Typography variant="caption" color="text.disabled" component="div">
                                  Entered by <CreatorName id={event.creatorId} variant="caption" /> · {formatTime(event.creationTime)}
                                </Typography>
                              </TimelineContent>
                            </TimelineItem>
                          ))}
                        </Timeline>
                      </CardContent>
                    </Card>
                  </Stack>

                  <Box sx={{ flex: 2, width: '100%', position: { md: 'sticky' }, top: { md: 88 } }}>
                    <StatusUpdate trackingNumber={details.trackingNumber} statuses={statuses} />
                  </Box>
                </Stack>
              )
        }
      </Box>
    </Box>
  )
}
