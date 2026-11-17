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

const SORT_FIELDS = ['trackingNumber', 'pol', 'pod', 'lastStatusOccurringTime', 'creationTime'] as const
type SortField = typeof SORT_FIELDS[number]
type SortDirection = 'asc' | 'desc'

const PAGE_SIZES = [10, 20, 50, 100] as const
type PageSize = typeof PAGE_SIZES[number]

// Every field is optional: omitted fields take the server-side default.
type TrackingListQuery = {
  trackingNumber?: string,
  pol?: string,
  pod?: string,
  creationTimeMin?: Time,
  creationTimeMax?: Time,
  lastStatusOccurringTimeMin?: Time,
  lastStatusOccurringTimeMax?: Time,
  sortBy?: SortField,
  sortDirection?: SortDirection,
  page?: number,
  pageSize?: PageSize,
}

function defaultSortDirection(sortBy: SortField): SortDirection {
  switch (sortBy) {
    case 'trackingNumber':
    case 'pol':
    case 'pod':
      return 'asc'
    case 'lastStatusOccurringTime':
    case 'creationTime':
      return 'desc'
  }
}

// ASSUMED (unspecified in api-shape.md): GET /api/admin/tracking/:trackingNumber
// Mirrors the customer TrackingResponse, but with admin-only metadata.
type TrackingDetails = Omit<Tracking, 'lastStatus'>

type AdminTrackingResponse = {
  details: TrackingDetails,  // last status is the last entry of `statuses`
  statuses: Status[],        // oldest first: occurringTime ASC, ties by creationTime ASC
}

// ASSUMED (unspecified): POST /api/admin/tracking/:trackingNumber/status
// The server fills in `id`, `creatorId` and `creationTime`; responds with the created Status.
type StatusCreate = (GoodStatus | BadStatus) & {
  occurringTime: Time,
  location: string,
}

export { GOOD_STATUS_NAMES, PAGE_SIZES, POSSIBLE_STATUSES, SORT_FIELDS, defaultSortDirection }
export type {
  AdminTrackingResponse, BadStatus, GoodStatus, GoodStatusName, PageSize, SortDirection, SortField,
  Status, StatusCreate, StatusCustomer, Tracking, TrackingBase, TrackingDetails, TrackingListQuery,
  TrackingListResponse,
}
