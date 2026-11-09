import { http, HttpResponse } from 'msw'
import { base } from '@/lib/api'

export const handlers = [
  http.get(`${base}/tracking/BLJ-good-code`, () => {
    return HttpResponse.json({
      details: {
        billNumber: 'goodcode',
        contOrSeal: 'abcd1234',
        pol: 'SomeRandomPort of Vietnam',
        pod: 'SomeRandomPort of Singapore'
      },
      statuses: [
        {
          time: '2026-10-15T09:11:00.000Z',
          title: 'Package resigered',
          warnings: [],
        },
        {
          time: '2026-10-15T09:11:30.000Z',
          title: 'Inland Trucking',
          location: 'Hanoi, VN',
          warnings: [],
        },
        {
          time: '2026-10-15T09:14:00.000Z',
          title: 'Customs Clearance',
          location: 'Hanoi, VN',
          warnings: [],
        },
        {
          time: '2026-10-20T09:15:00.000Z',
          title: 'Arrived at Port of Loading',
          location: 'Hanoi, VN',
          warnings: [],
        },
        {
          time: '2026-10-21T14:30:00.000Z',
          title: 'Left Port of Loading',
          location: 'Hanoi, VN',
          warnings: [],
        },
        {
          time: '2026-10-23T08:00:00.000Z',
          title: 'Arrived at Port of Discharge',
          location: 'Singapore, Singapore',
          warnings: [],
        },
        {
          time: '2026-10-24T11:45:00.000Z',
          title: 'Customs Clearance',
          location: 'Singapore, Singapore',
          warnings: ['Delayed by customs inspection'],
        },
        {
          time: '2026-10-24T11:45:00.000Z',
          title: 'Inland Trucking to Place of Delivery',
          location: 'Singapore, Singapore',
          warnings: [],
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
