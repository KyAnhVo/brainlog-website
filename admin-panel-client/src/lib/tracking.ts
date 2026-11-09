const POSSIBLE_STATUSES = new Set([
  'Tracking created', 'Ex-work', 'Customs clearance at POL', 'Arrived at POL', 'Left POL',
  'Arrived at POT', 'Left POT', 'Arrived at POD', 'Left POD',
  'Customs clearance at POD', 'Inland trucking to Place of Delivery',
  'Arrived at Place of Delivery', 'Finished'
])

type TrackingDetails = {
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

type StatusEvent = {
  type: 'good',
  status: string,
  occurringTime: Date,
  location: string,
  warnings: string[],
  creationTime: Date,
  statusCreator: string,
}

type StatusErrorEvent = {
  type: 'bad',
  status: string,
  occurringTime: Date,
  location: string,
  creationTime: Date,
  statusCreator: string,
}

type TrackingEvent = StatusEvent | StatusErrorEvent

// At least an id, so the panel can always link to /customer/$customerId.
type CustomerRef = {
  id: string,
  name?: string,
  email?: string,
  phone?: string,
}

type TrackingOverview = TrackingDetails & {
  trackingCreator: string,
  trackingCreationTime: Date,
  latestStatus: TrackingEvent,
  customer?: CustomerRef,
}

export {
  type TrackingDetails, type StatusEvent, type StatusErrorEvent, type TrackingEvent,
  type CustomerRef, type TrackingOverview, POSSIBLE_STATUSES,
}
