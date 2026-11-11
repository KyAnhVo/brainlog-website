import type { Time } from '@/lib/time'

const GOOD_STATUS_NAMES = [
  'Tracking created', 'Ex-work', 'Customs clearance at POL', 'Arrived at POL',
  'Left POL', 'Arrived at POT', 'Left POT', 'Arrived at POD', 'Left POD',
  'Customs clearance at POD', 'Inland trucking to Place of Delivery',
  'Arrived at Place of Delivery', 'Finished',
] as const

type GoodStatusName = typeof GOOD_STATUS_NAMES[number]

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

// GET /api/tracking/:trackingNumber
type TrackingResponse = {
  details: TrackingBase,
  statuses: StatusCustomer[],  // ordered by occurringTime, oldest first
}

export type { GoodStatusName, StatusCustomer, TrackingBase, TrackingResponse }
