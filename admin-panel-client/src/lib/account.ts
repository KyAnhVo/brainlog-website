import type { Time } from '@/lib/time'

type Role = 'admin' | 'user'

// POST /api/admin/login
type Login = {
  email: string,
  password: string,
}

type LoginSuccessResponse = {
  name: string,
  account_created: Time,
  role: Role,
}

// POST /api/admin/createAccount
type AccountCreate = {
  email: string,
  password: string,
  name: string,
  role: Role,
}

// ASSUMED path: POST /api/admin/deleteAccount
type AccountDelete = { email: string } | { id: string }

// ASSUMED (unspecified): GET /api/admin/account/:id, resolves a creatorId to a display name.
type AccountPublic = {
  id: string,
  name: string,
}

export type { AccountCreate, AccountDelete, AccountPublic, Login, LoginSuccessResponse, Role }
