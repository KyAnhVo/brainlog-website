// Mock "database" for the admin panel. Documented in /devnote/admin-mock.md.
// Kept free of browser/MSW imports so it can also be run by bun for doc generation.
import type { Role } from '@/lib/account'
import type { GoodStatusName, Status, TrackingDetails } from '@/lib/tracking'
import type { Time } from '@/lib/time'

type MockAccount = {
  id: string,
  email: string,
  password: string,  // plaintext ONLY because this is a mock; the real server stores Argon2id hashes
  name: string,
  account_create_time: Time,
  role: Role,
}

type MockTracking = {
  details: TrackingDetails,
  statuses: Status[],  // kept sorted oldest first
}

// "2026-10-15 09:11" -> "2026-10-15T09:11:00.000Z"
function t(s: string): Time {
  return `${s.replace(' ', 'T')}:00.000Z`
}

const ALICE = '0199a3f0-1c2e-7a41-9b3e-5d2f8c1a0e01'
const BINH = '0199a3f0-4d7b-7c12-8f60-2b9e4a7d1c02'
const CHLOE = '0199a3f1-0a93-7e55-a1c4-8e3f6b2d9a03'
const DUNG = '0199a3f1-77c2-7f08-b2d1-4c5e9f1a6b04'
// Account that created old data, then was deleted: name lookup returns 404.
const DELETED = '01989e2c-5b10-7d3a-9e77-0f1a2b3c4d99'

const accounts: MockAccount[] = [
  { id: ALICE, email: 'admin@brainlog.test', password: 'Admin123!', name: 'Alice Nguyen', account_create_time: t('2026-01-05 08:00'), role: 'admin' },
  { id: BINH, email: 'staff@brainlog.test', password: 'Staff123!', name: 'Bình Trần', account_create_time: t('2026-02-11 02:30'), role: 'user' },
  { id: CHLOE, email: 'ops@brainlog.test', password: 'Ops12345!', name: 'Chloé Martin', account_create_time: t('2026-03-20 14:15'), role: 'user' },
  { id: DUNG, email: 'dung@brainlog.test', password: 'Dung1234!', name: 'Dũng Phạm', account_create_time: t('2026-06-01 09:45'), role: 'admin' },
]

// One status in a tracking's history:
//   [status, occurringTime, location, warnings?, creatorId?, minutesUntilEntered?]
// `status` is a good status name, or `{ bad: '...' }`.
// creationTime = occurringTime + minutesUntilEntered (default 15).
type Step = [
  GoodStatusName | { bad: string },
  string,
  string,
  string[]?,
  string?,
  number?,
]

let statusSeq = 0

function build(details: TrackingDetails, steps: Step[]): MockTracking {
  const statuses: Status[] = steps.map(([status, occ, location, warnings = [], creatorId = details.creatorId, delayMin = 15]) => {
    const occurringTime = t(occ)
    const creationTime = new Date(Date.parse(occurringTime) + delayMin * 60_000).toISOString()
    const id = `status-${++statusSeq}`
    return typeof status === 'string'
      ? { id, statusType: 'good', status, warnings, occurringTime, location, creatorId, creationTime }
      : { id, statusType: 'bad', status: status.bad, occurringTime, location, creatorId, creationTime }
  })
  return { details, statuses: sortStatuses(statuses) }
}

// occurringTime ASC, ties by creationTime ASC
function sortStatuses(statuses: Status[]): Status[] {
  return [...statuses].sort((a, b) =>
    a.occurringTime.localeCompare(b.occurringTime) || a.creationTime.localeCompare(b.creationTime))
}

const VN_HCM = 'Hồ Chí Minh, VN'
const VN_HP = 'Hải Phòng, VN'
const VN_DN = 'Đà Nẵng, VN'
const SG = 'Singapore, SG'
const KR = 'Busan, KR'
const CN = 'Shanghai, CN'
const JP = 'Tokyo, JP'
const US = 'Los Angeles, US'
const NL = 'Rotterdam, NL'
const DE = 'Hamburg, DE'
const MY = 'Port Klang, MY'
const TH = 'Laem Chabang, TH'

const trackings: MockTracking[] = [
  // Same tracking numbers as the customer-facing mocks.
  build({
    trackingNumber: 'BLJ-good-code', cont: 'MSKU1234567', seal: 'SL998877', pol: VN_HP, pod: SG,
    physicalDetails: { pcsCount: 12, weightKg: 840.5, dimensions: { lengthCm: 240, widthCm: 120, heightCm: 150 } },
    creatorId: ALICE, creationTime: t('2026-10-15 09:11'), customerId: 'customer-1',
  }, [
    ['Tracking created', '2026-10-15 09:11', VN_HP, [], ALICE, 0],
    ['Customs clearance at POL', '2026-10-15 09:14', VN_HP],
    ['Arrived at POL', '2026-10-20 09:15', VN_HP, [], BINH],
    ['Left POL', '2026-10-21 14:30', VN_HP, [], BINH],
    ['Arrived at POD', '2026-10-23 08:00', SG, [], CHLOE],
    ['Customs clearance at POD', '2026-10-24 11:45', SG, ['Delayed by customs inspection'], CHLOE, 10],
    // Same occurringTime as the previous status: entered later, so it is the last status.
    ['Inland trucking to Place of Delivery', '2026-10-24 11:45', SG, [], CHLOE, 20],
  ]),
  build({
    trackingNumber: 'BLJ-good-code-fail-shipment', pol: VN_DN, pod: KR,
    creatorId: ALICE, creationTime: t('2026-10-15 09:11'),
  }, [
    ['Tracking created', '2026-10-15 09:11', VN_DN, [], ALICE, 0],
    ['Arrived at POL', '2026-10-20 09:15', VN_DN, [], BINH],
    ['Customs clearance at POL', '2026-10-21 10:00', VN_DN, ['Additional documents requested by customs'], BINH],
    [{ bad: 'Rejected by customs: prohibited item' }, '2026-10-22 08:00', VN_DN, [], ALICE],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0001', cont: 'TGHU8821340', seal: 'HL220031', pol: VN_HCM, pod: US,
    physicalDetails: { pcsCount: 40, weightKg: 18250, dimensions: { lengthCm: 1203, widthCm: 235, heightCm: 239 } },
    creatorId: BINH, creationTime: t('2026-09-02 01:05'), customerId: 'customer-2',
  }, [
    ['Tracking created', '2026-09-02 01:05', VN_HCM, [], BINH, 0],
    ['Ex-work', '2026-09-03 07:00', 'Bình Dương, VN'],
    ['Customs clearance at POL', '2026-09-05 03:20', VN_HCM],
    ['Arrived at POL', '2026-09-05 10:00', VN_HCM],
    ['Left POL', '2026-09-06 22:40', VN_HCM],
    ['Arrived at POT', '2026-09-09 05:10', 'Kaohsiung, TW', [], CHLOE],
    ['Left POT', '2026-09-10 18:00', 'Kaohsiung, TW', [], CHLOE],
    ['Arrived at POD', '2026-09-28 16:30', US, [], CHLOE],
    ['Customs clearance at POD', '2026-09-30 19:00', US, [], CHLOE],
    ['Inland trucking to Place of Delivery', '2026-10-01 15:00', 'Ontario, CA, US', [], CHLOE],
    ['Arrived at Place of Delivery', '2026-10-02 21:15', 'Ontario, CA, US', [], CHLOE],
    ['Finished', '2026-10-03 00:00', 'Ontario, CA, US', [], CHLOE],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0002', cont: 'CMAU5530912', pol: VN_HCM, pod: NL,
    physicalDetails: { weightKg: 9600, dimensions: { lengthCm: 590, widthCm: 235, heightCm: 239 } },
    creatorId: BINH, creationTime: t('2026-09-14 03:00'),
  }, [
    ['Tracking created', '2026-09-14 03:00', VN_HCM, [], BINH, 0],
    ['Arrived at POL', '2026-09-16 08:00', VN_HCM],
    ['Left POL', '2026-09-18 12:00', VN_HCM],
    ['Arrived at POT', '2026-09-22 06:00', SG, ['Transshipment window extended by 48h']],
    ['Left POT', '2026-09-25 09:30', SG],
    ['Arrived at POD', '2026-10-19 04:45', NL, [], CHLOE],
    ['Customs clearance at POD', '2026-10-21 11:00', NL, [], CHLOE],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0003', pol: VN_DN, pod: JP,
    physicalDetails: { pcsCount: 3, weightKg: 120.25, dimensions: { lengthCm: 80, widthCm: 60, heightCm: 45 } },
    creatorId: CHLOE, creationTime: t('2026-10-01 06:30'), customerId: 'customer-3',
  }, [
    ['Tracking created', '2026-10-01 06:30', VN_DN, [], CHLOE, 0],
    ['Ex-work', '2026-10-02 02:00', 'Hội An, VN'],
    ['Arrived at POL', '2026-10-03 09:00', VN_DN],
    ['Left POL', '2026-10-04 13:00', VN_DN],
    ['Arrived at POD', '2026-10-08 01:00', JP],
    ['Customs clearance at POD', '2026-10-08 07:30', JP],
    ['Arrived at Place of Delivery', '2026-10-09 05:00', 'Yokohama, JP'],
    ['Finished', '2026-10-09 05:30', 'Yokohama, JP'],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0004', cont: 'OOLU7781203', seal: 'OO551200', pol: CN, pod: VN_HP,
    creatorId: ALICE, creationTime: t('2026-10-05 10:00'),
  }, [
    ['Tracking created', '2026-10-05 10:00', CN, [], ALICE, 0],
    ['Arrived at POL', '2026-10-06 04:00', CN],
    ['Left POL', '2026-10-07 20:00', CN],
    [{ bad: 'Vessel diverted due to typhoon' }, '2026-10-09 03:00', 'South China Sea', [], DUNG],
    ['Arrived at POD', '2026-10-14 12:00', VN_HP, ['Arrived 4 days late'], DUNG],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0005', pol: SG, pod: VN_HCM,
    physicalDetails: { pcsCount: 1, weightKg: 2.5, dimensions: { lengthCm: 30, widthCm: 20, heightCm: 10 } },
    creatorId: DUNG, creationTime: t('2026-10-10 02:15'),
  }, [
    ['Tracking created', '2026-10-10 02:15', SG, [], DUNG, 0],
    ['Arrived at POL', '2026-10-11 03:00', SG],
    ['Left POL', '2026-10-11 15:00', SG],
    ['Arrived at POD', '2026-10-13 06:00', VN_HCM],
    ['Customs clearance at POD', '2026-10-14 02:00', VN_HCM, ['Duty unpaid, awaiting consignee']],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0006', cont: 'HLXU2209981', pol: VN_HP, pod: DE,
    physicalDetails: { pcsCount: 220, weightKg: 21400, dimensions: { lengthCm: 1203, widthCm: 235, heightCm: 269 } },
    creatorId: DELETED, creationTime: t('2026-08-20 07:00'), customerId: 'customer-4',
  }, [
    ['Tracking created', '2026-08-20 07:00', VN_HP, [], DELETED, 0],
    ['Arrived at POL', '2026-08-22 05:00', VN_HP, [], DELETED],
    ['Left POL', '2026-08-23 10:00', VN_HP, [], DELETED],
    ['Arrived at POT', '2026-08-27 08:00', SG, [], DELETED],
    ['Left POT', '2026-08-28 23:00', SG, [], DELETED],
    ['Arrived at POD', '2026-09-24 06:00', DE, [], ALICE],
    ['Customs clearance at POD', '2026-09-25 09:00', DE, [], ALICE],
    ['Inland trucking to Place of Delivery', '2026-09-26 05:00', 'Berlin, DE', [], ALICE],
    ['Arrived at Place of Delivery', '2026-09-27 13:00', 'Berlin, DE', [], ALICE],
    ['Finished', '2026-09-27 14:00', 'Berlin, DE', [], ALICE],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0007', pol: MY, pod: VN_DN,
    creatorId: BINH, creationTime: t('2026-10-25 04:40'),
  }, [
    ['Tracking created', '2026-10-25 04:40', MY, [], BINH, 0],
    ['Arrived at POL', '2026-10-27 02:00', MY],
    ['Left POL', '2026-10-28 06:00', MY],
    ['Arrived at POD', '2026-10-31 10:00', VN_DN],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0008', cont: 'EGLV1100293', seal: 'EG093311', pol: TH, pod: VN_HCM,
    physicalDetails: { pcsCount: 18, weightKg: 4410, dimensions: { lengthCm: 590, widthCm: 235, heightCm: 239 } },
    creatorId: CHLOE, creationTime: t('2026-10-28 08:20'),
  }, [
    ['Tracking created', '2026-10-28 08:20', TH, [], CHLOE, 0],
    ['Customs clearance at POL', '2026-10-29 03:00', TH],
    [{ bad: 'Container damaged during loading' }, '2026-10-30 07:10', TH],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0009', pol: VN_HCM, pod: SG,
    physicalDetails: { pcsCount: 6, weightKg: 310, dimensions: { lengthCm: 120, widthCm: 100, heightCm: 90 } },
    creatorId: ALICE, creationTime: t('2026-11-02 01:00'), customerId: 'customer-1',
  }, [
    ['Tracking created', '2026-11-02 01:00', VN_HCM, [], ALICE, 0],
    ['Ex-work', '2026-11-03 02:00', 'Long An, VN'],
    ['Arrived at POL', '2026-11-04 04:30', VN_HCM, [], BINH],
    ['Left POL', '2026-11-05 11:00', VN_HCM, [], BINH],
    ['Arrived at POD', '2026-11-07 09:00', SG, [], BINH],
    // Entered 2 days late (backfilled).
    ['Customs clearance at POD', '2026-11-08 03:00', SG, [], CHLOE, 2 * 24 * 60],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0010', cont: 'MSCU4432019', pol: KR, pod: VN_HP,
    creatorId: DUNG, creationTime: t('2026-11-03 05:00'),
  }, [
    ['Tracking created', '2026-11-03 05:00', KR, [], DUNG, 0],
    ['Arrived at POL', '2026-11-04 01:00', KR],
    ['Left POL', '2026-11-05 08:00', KR],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0011', pol: VN_HP, pod: CN,
    physicalDetails: { weightKg: 1500, dimensions: { lengthCm: 300, widthCm: 200, heightCm: 180 } },
    creatorId: BINH, creationTime: t('2026-11-05 03:30'),
  }, [
    ['Tracking created', '2026-11-05 03:30', VN_HP, [], BINH, 0],
    ['Customs clearance at POL', '2026-11-06 02:00', VN_HP, ['Inspection selected (red channel)', 'Missing packing list']],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0012', cont: 'ZIMU3319028', seal: 'ZM112233', pol: VN_HCM, pod: US,
    physicalDetails: { pcsCount: 960, weightKg: 19800, dimensions: { lengthCm: 1203, widthCm: 235, heightCm: 269 } },
    creatorId: ALICE, creationTime: t('2026-11-06 06:00'), customerId: 'customer-2',
  }, [
    ['Tracking created', '2026-11-06 06:00', VN_HCM, [], ALICE, 0],
    ['Ex-work', '2026-11-07 01:00', 'Đồng Nai, VN'],
    ['Arrived at POL', '2026-11-08 08:00', VN_HCM],
    ['Left POL', '2026-11-10 17:00', VN_HCM],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0013', pol: JP, pod: VN_DN,
    creatorId: CHLOE, creationTime: t('2026-11-08 00:30'),
  }, [
    ['Tracking created', '2026-11-08 00:30', JP, [], CHLOE, 0],
    ['Arrived at POL', '2026-11-09 03:00', JP],
    [{ bad: 'Shipment held: consignee unreachable' }, '2026-11-11 08:00', JP],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0014', pol: VN_DN, pod: SG,
    physicalDetails: { pcsCount: 2, weightKg: 48, dimensions: { lengthCm: 60, widthCm: 40, heightCm: 40 } },
    creatorId: DUNG, creationTime: t('2026-11-10 09:00'),
  }, [
    ['Tracking created', '2026-11-10 09:00', VN_DN, [], DUNG, 0],
    ['Arrived at POL', '2026-11-11 02:00', VN_DN],
    ['Left POL', '2026-11-12 05:00', VN_DN],
    ['Arrived at POD', '2026-11-14 07:00', SG],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0015', cont: 'TCLU6650021', pol: NL, pod: VN_HCM,
    creatorId: ALICE, creationTime: t('2026-10-12 12:00'),
  }, [
    ['Tracking created', '2026-10-12 12:00', NL, [], ALICE, 0],
    ['Arrived at POL', '2026-10-14 08:00', NL],
    ['Left POL', '2026-10-15 22:00', NL],
    ['Arrived at POT', '2026-11-01 04:00', 'Colombo, LK'],
    ['Left POT', '2026-11-03 10:00', 'Colombo, LK'],
    ['Arrived at POD', '2026-11-12 06:00', VN_HCM, [], BINH],
    ['Customs clearance at POD', '2026-11-13 03:00', VN_HCM, [], BINH],
    ['Inland trucking to Place of Delivery', '2026-11-14 01:00', 'Bình Dương, VN', [], BINH],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0016', pol: VN_HP, pod: KR,
    creatorId: BINH, creationTime: t('2026-11-12 02:00'),
  }, [
    ['Tracking created', '2026-11-12 02:00', VN_HP, [], BINH, 0],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0017', cont: 'SEGU9001234', seal: 'SE776655', pol: SG, pod: DE,
    physicalDetails: { pcsCount: 75, weightKg: 12040.75, dimensions: { lengthCm: 1203, widthCm: 235, heightCm: 239 } },
    creatorId: CHLOE, creationTime: t('2026-11-13 04:00'),
  }, [
    ['Tracking created', '2026-11-13 04:00', SG, [], CHLOE, 0],
    ['Arrived at POL', '2026-11-14 06:00', SG],
    ['Left POL', '2026-11-15 13:00', SG],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0018', pol: VN_HCM, pod: TH,
    creatorId: DUNG, creationTime: t('2026-11-14 08:00'), customerId: 'customer-3',
  }, [
    ['Tracking created', '2026-11-14 08:00', VN_HCM, [], DUNG, 0],
    ['Ex-work', '2026-11-15 02:00', 'Thủ Đức, VN', ['Pickup rescheduled once']],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0019', pol: CN, pod: VN_HCM,
    physicalDetails: { pcsCount: 10, weightKg: 560, dimensions: { lengthCm: 200, widthCm: 120, heightCm: 100 } },
    creatorId: ALICE, creationTime: t('2026-11-15 03:00'),
  }, [
    ['Tracking created', '2026-11-15 03:00', CN, [], ALICE, 0],
    ['Arrived at POL', '2026-11-16 01:00', CN],
  ]),
  build({
    trackingNumber: 'BLJ-2026-0020', pol: VN_DN, pod: MY,
    creatorId: BINH, creationTime: t('2026-11-16 07:30'),
  }, [
    ['Tracking created', '2026-11-16 07:30', VN_DN, [], BINH, 0],
  ]),
  build({
    trackingNumber: 'BLX-SAMPLE-0021', pol: US, pod: VN_HP,
    creatorId: CHLOE, creationTime: t('2026-09-20 18:00'),
  }, [
    ['Tracking created', '2026-09-20 18:00', US, [], CHLOE, 0],
    ['Arrived at POL', '2026-09-22 16:00', US],
    ['Left POL', '2026-09-24 02:00', US],
    ['Arrived at POT', '2026-10-08 03:00', 'Kaohsiung, TW'],
    [{ bad: 'Lost in transit: investigation opened' }, '2026-10-12 05:00', 'Kaohsiung, TW', [], DUNG],
  ]),
  build({
    trackingNumber: 'BLX-SAMPLE-0022', pol: KR, pod: SG,
    creatorId: DUNG, creationTime: t('2026-10-18 01:00'),
  }, [
    ['Tracking created', '2026-10-18 01:00', KR, [], DUNG, 0],
    ['Arrived at POL', '2026-10-19 05:00', KR],
    ['Left POL', '2026-10-20 09:00', KR],
    ['Arrived at POD', '2026-10-24 05:00', SG],
    ['Customs clearance at POD', '2026-10-24 12:00', SG],
    ['Arrived at Place of Delivery', '2026-10-25 08:00', 'Jurong, SG'],
    ['Finished', '2026-10-25 09:00', 'Jurong, SG'],
  ]),
]

export { accounts, sortStatuses, trackings }
export type { MockAccount, MockTracking }
