import { http, HttpResponse } from 'msw'
import { base } from '@/lib/api'
import type { TrackingListResponse } from '@/lib/tracking'

const trackingList: TrackingListResponse = {
  trackings: [
    {
      trackingNumber: 'BLJ-good-code',
      cont: 'MSKU1234567',
      seal: 'SL998877',
      pol: 'Hai Phong, VN',
      pod: 'Singapore, SG',
      physicalDetails: {
        pcsCount: 12,
        weightKg: 840.5,
        dimensions: { lengthCm: 240, widthCm: 120, heightCm: 150 },
      },
      creatorId: 'user-1',
      creationTime: '2026-10-15T09:11:00.000Z',
      customerId: 'customer-1',
      lastStatus: {
        id: 'status-7',
        statusType: 'good',
        status: 'Customs clearance at POD',
        occurringTime: '2026-10-24T11:45:00.000Z',
        location: 'Singapore, SG',
        warnings: ['Delayed by customs inspection'],
        creatorId: 'user-2',
        creationTime: '2026-10-24T12:02:13.000Z',
      },
    },
    {
      trackingNumber: 'BLJ-good-code-fail-shipment',
      pol: 'Da Nang, VN',
      pod: 'Busan, KR',
      creatorId: 'user-1',
      creationTime: '2026-10-18T03:20:00.000Z',
      lastStatus: {
        id: 'status-12',
        statusType: 'bad',
        status: 'Cargo rejected by customs',
        occurringTime: '2026-10-22T07:30:00.000Z',
        location: 'Busan, KR',
        creatorId: 'user-3',
        creationTime: '2026-10-22T08:00:00.000Z',
      },
    },
  ],
  page: 1,
  pageCount: 1,
  pageSize: 20,
}

export const handlers = [
  http.get(`${base}/admin/tracking`, () => {
    return HttpResponse.json(trackingList)
  }),
]
