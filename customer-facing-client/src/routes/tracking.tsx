import { Box, Card, CardContent, CardHeader, Typography } from '@mui/material';
import { createFileRoute, Outlet, useNavigate } from '@tanstack/react-router'
import React, { useState } from 'react';

import logo from '@/assets/logo.png';

import TrackingLookup from './tracking/-trackingLookup.tsx'
export const Route = createFileRoute('/tracking')({
  component: RouteComponent,
})

function RouteComponent() {
  const [trackingNumber, setTrackingNumber] = useState('');
  const navigate = useNavigate();

  const handleSubmit = (e: React.SubmitEvent) => {
    e.preventDefault();
    navigate({
      to: '/tracking/$trackingNumber',
      params: { trackingNumber },
    })
  }

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
              Tracking
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
          <TrackingLookup
            setTrackingNumber={setTrackingNumber}
            handleSubmit={handleSubmit}
          />
          <Outlet />
        </CardContent>
      </Card>
    </Box>
  )
}
