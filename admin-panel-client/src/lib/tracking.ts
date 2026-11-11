import type { Time } from '@/lib/time'

const GOOD_STATUS_NAMES = [
  'Tracking created', 'Ex-work', 'Customs clearance at POL', 'Arrived at POL',
  'Left POL', 'Arrived at POT', 'Left POT', 'Arrived at POD', 'Left POD',
  'Customs clearance at POD', 'Inland trucking to Place of Delivery',
  'Arrived at Place of Delivery', 'Finished',
] as const

type GoodStatusName = typeof GOOD_STATUS_NAMES[number]

const POSSIBLE_STATUSES: ReadonlySet<string> = new Set(GOOD_STATUS_NAMES)

type GoodStatus = {
  statusType: 'good',
  status: GoodStatusName,
  warnings: string[],  // empty array when none, never omitted
}

type BadStatus = {
  statusType: 'bad',
  status: string,  // nonempty
}

type StatusCustomer = (GoodStatus | BadStatus) & {
  id: string,
  occurringTime: Time,
  location: string,
}

type Status = StatusCustomer & {
  creatorId: string,
  creationTime: Time,
}

type TrackingBase = {
  trackingNumber: string,
  cont?: string,
  seal?: string,
  pol: string,
  pod: string,
  physicalDetails?: {
    pcsCount?: number,
    weightKg: number,
    dimensions: {
      lengthCm: number,
      widthCm: number,
      heightCm: number,
    },
  },
}

type Tracking = TrackingBase & {
  creatorId: string,
  creationTime: Time,
  lastStatus: Status,
  customerId?: string,
}

// GET /api/admin/tracking
type TrackingListResponse = {
  trackings: Tracking[],
  page: number,
  pageCount: number,
  pageSize: number,
}

export { POSSIBLE_STATUSES }
export type { GoodStatusName, Status, Tracking, TrackingListResponse }
