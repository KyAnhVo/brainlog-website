# Note: **please read this first**

# Syntax for type

All type/shape definitions use typescript syntax:

```typescript
type X = {...}
```

This does not imply that this must be on the front-end. We will specify if something is needed for the front-end or the backend.

This can be a Java class, a TS class, a TS type, etc.

# Time

 Time is sent/received and stored at UTC time, `RFC 3339` with `Z` suffix and millisecond precision, both in JSON body and query parameters.  The shape MUST BE `yyyy-mm-ddThh:mm:ss.xxxZ`. Even if `xxx == 000`, `000` must still be there.

```typescript
type Time = string;
function isValidTime(s: string): boolean
```

`isValidTime` rejects anything not exactly of that shape, including out-of-range fields that `Date.parse` would roll over (e.g. `2026-02-30T00:00:00.000Z`).
