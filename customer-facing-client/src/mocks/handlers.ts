import { http, HttpResponse } from 'msw'
import { base } from '@/lib/api'

export const handlers = [
  http.get(`${base}/tracking/BLJ-good-code`, () => {
    return HttpResponse.json({
      details: {
        trackingNumber: 'BLJ-good-code',
        contOrSeal: 'abcd1234',
        pol: 'SomeRandomPort of Vietnam',
        pod: 'SomeRandomPort of Singapore',
        physicalDetails: {
          pieces: 12,
          weightKg: 840.5,
          dimensionsM: { length: 2.4, width: 1.2, height: 1.5 },
        },
      },
      statuses: [
        {
          type: 'good',
          status: 'Package registered',
          occurringTime: '2026-10-15T09:11:00.000Z',
          location: 'Hanoi, VN',
          warnings: [],
        },
        {
          type: 'good',
          status: 'Inland Trucking',
          occurringTime: '2026-10-15T09:11:30.000Z',
          location: 'Hanoi, VN',
          warnings: [],
        },
        {
          type: 'good',
          status: 'Customs Clearance',
          occurringTime: '2026-10-15T09:14:00.000Z',
          location: 'Hanoi, VN',
          warnings: [],
        },
        {
          type: 'good',
          status: 'Arrived at Port of Loading',
          occurringTime: '2026-10-20T09:15:00.000Z',
          location: 'Hanoi, VN',
          warnings: [],
        },
        {
          type: 'good',
          status: 'Left Port of Loading',
          occurringTime: '2026-10-21T14:30:00.000Z',
          location: 'Hanoi, VN',
          warnings: [],
        },
        {
          type: 'good',
          status: 'Arrived at Port of Discharge',
          occurringTime: '2026-10-23T08:00:00.000Z',
          location: 'Singapore, Singapore',
          warnings: [],
        },
        {
          type: 'good',
          status: 'Customs Clearance',
          occurringTime: '2026-10-24T11:45:00.000Z',
          location: 'Singapore, Singapore',
          warnings: ['Delayed by customs inspection'],
        },
        {
          type: 'good',
          status: 'Inland Trucking to Place of Delivery',
          occurringTime: '2026-10-24T11:45:00.000Z',
          location: 'Singapore, Singapore',
          warnings: [],
        },
      ]
    })
  }),

  http.get(`${base}/tracking/BLJ-good-code-fail-shipment`, () => {
    return HttpResponse.json({
      details: {
        trackingNumber: 'BLJ-good-code-fail-shipment',
        pol: 'SomeRandomPort of Vietnam',
        pod: 'SomeRandomPort of Singapore',
        physicalDetails: {
          pieces: 3,
        },
      },
      statuses: [
        {
          type: 'good',
          status: 'Package registered',
          occurringTime: '2026-10-15T09:11:00.000Z',
          location: 'Hanoi, VN',
          warnings: [],
        },
        {
          type: 'good',
          status: 'Arrived at Port of Loading',
          occurringTime: '2026-10-20T09:15:00.000Z',
          location: 'Hanoi, VN',
          warnings: [],
        },
        {
          type: 'good',
          status: 'Customs Clearance',
          occurringTime: '2026-10-21T10:00:00.000Z',
          location: 'Hanoi, VN',
          warnings: ['Additional documents requested by customs'],
        },
        {
          type: 'bad',
          status: 'Rejected by customs: prohibited item',
          occurringTime: '2026-10-22T08:00:00.000Z',
          location: 'Hanoi, VN',
        },
      ]
    })
  }),

  http.get(`${base}/tracking/BLJ-badd-code`, () => {
    return HttpResponse.json(
      { errorCode: 404, errorMsg: 'No shipment found for this tracking code' },
      { status: 404 },
    )
  }),
]
