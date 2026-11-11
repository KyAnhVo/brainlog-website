// UTC, RFC 3339, millisecond precision, `Z` suffix: yyyy-mm-ddThh:mm:ss.xxxZ
type Time = string

const TIME_SHAPE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/

// Date.parse rolls out-of-range fields over (02-30 -> 03-02, 24:00 -> next day),
// so the round trip must reproduce the input exactly.
function isValidTime(s: string): boolean {
  if (!TIME_SHAPE.test(s)) return false
  const t = Date.parse(s)
  return !Number.isNaN(t) && new Date(t).toISOString() === s
}

export { isValidTime }
export type { Time }
