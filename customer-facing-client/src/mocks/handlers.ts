import { http, HttpResponse } from 'msw'
import { base } from '@/lib/api'
import type { TrackingResponse } from '@/lib/tracking'

const goodCode: TrackingResponse = {
  details: {
    trackingNumber: 'BLJ-good-code',
    cont: 'MSKU1234567',
    seal: 'SL998877',
    pol: 'SomeRandomPort of Vietnam',
    pod: 'SomeRandomPort of Singapore',
    physicalDetails: {
      pcsCount: 12,
      weightKg: 840.5,
      dimensions: { lengthCm: 240, widthCm: 120, heightCm: 150 },
    },
  },
  statuses: [
    {
      id: 'status-1',
      statusType: 'good',
      status: 'Tracking created',
      occurringTime: '2026-10-15T09:11:00.000Z',
      location: 'Hanoi, VN',
      warnings: [],
    },
    {
      id: 'status-2',
      statusType: 'good',
      status: 'Customs clearance at POL',
      occurringTime: '2026-10-15T09:14:00.000Z',
      location: 'Hanoi, VN',
      warnings: [],
    },
    {
      id: 'status-3',
      statusType: 'good',
      status: 'Arrived at POL',
      occurringTime: '2026-10-20T09:15:00.000Z',
      location: 'Hanoi, VN',
      warnings: [],
    },
    {
      id: 'status-4',
      statusType: 'good',
      status: 'Left POL',
      occurringTime: '2026-10-21T14:30:00.000Z',
      location: 'Hanoi, VN',
      warnings: [],
    },
    {
      id: 'status-5',
      statusType: 'good',
      status: 'Arrived at POD',
      occurringTime: '2026-10-23T08:00:00.000Z',
      location: 'Singapore, Singapore',
      warnings: [],
    },
    {
      id: 'status-6',
      statusType: 'good',
      status: 'Customs clearance at POD',
      occurringTime: '2026-10-24T11:45:00.000Z',
      location: 'Singapore, Singapore',
      warnings: ['Delayed by customs inspection'],
    },
    {
      id: 'status-7',
      statusType: 'good',
      status: 'Inland trucking to Place of Delivery',
      occurringTime: '2026-10-24T11:45:00.000Z',
      location: 'Singapore, Singapore',
      warnings: [],
    },
  ],
}

const failShipment: TrackingResponse = {
  details: {
    trackingNumber: 'BLJ-good-code-fail-shipment',
    pol: 'SomeRandomPort of Vietnam',
    pod: 'SomeRandomPort of Singapore',
  },
  statuses: [
    {
      id: 'status-8',
      statusType: 'good',
      status: 'Tracking created',
      occurringTime: '2026-10-15T09:11:00.000Z',
      location: 'Hanoi, VN',
      warnings: [],
    },
    {
      id: 'status-9',
      statusType: 'good',
      status: 'Arrived at POL',
      occurringTime: '2026-10-20T09:15:00.000Z',
      location: 'Hanoi, VN',
      warnings: [],
    },
    {
      id: 'status-10',
      statusType: 'good',
      status: 'Customs clearance at POL',
      occurringTime: '2026-10-21T10:00:00.000Z',
      location: 'Hanoi, VN',
      warnings: ['Additional documents requested by customs'],
    },
    {
      id: 'status-11',
      statusType: 'bad',
      status: 'Rejected by customs: prohibited item',
      occurringTime: '2026-10-22T08:00:00.000Z',
      location: 'Hanoi, VN',
    },
  ],
}

export const handlers = [
  http.get(`${base}/tracking/BLJ-good-code`, () => {
    return HttpResponse.json(goodCode)
  }),

  http.get(`${base}/tracking/BLJ-good-code-fail-shipment`, () => {
    return HttpResponse.json(failShipment)
  }),

  http.get(`${base}/tracking/BLJ-badd-code`, () => {
    return HttpResponse.json(
      { errorCode: 404, errorMsg: 'No shipment found for this tracking code' },
      { status: 404 },
    )
  }),
]
