**Note** - All of the below is for the Admin site, as the customer site does not need login.
# Account 

Note: must be stored in back-end and/or DB, not front-end

An account for the admin site must have:

```ts
type Account = {
	id: string,
	email: string,
	password_hashed: string,
	name: string,
	account_create_time: Time,
	role: "admin" | "user"
};
```

## ID
`id` is the foreign key for the database. ID is generated via `UUIDv7` format.
## Email
There is no 2 accounts that share the same `id`. `id` is a candidate key.
## Password
The password must follow the standard from OWASP:
- **Use [Argon2id](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html#argon2id) with a  configuration of 19 MiB of memory, an iteration count of 2, and 1 degree of parallelism.**
# Login

An admin site must have a valid session cookie to enter anything that is not `/login`.

A cookie is valid for 7 days.

The API shape for Login is as following, which is sent to `POST /admin/login`:

```ts
type Login = {
	email: string,
	password: string,
}
```

If login is valid, we return `200`, a cookie with randomly generated 128 bits of session ID (which must be stored on the database, explained later):

```ts
type LoginSuccessResponse = {
	name: string,
	account_created: Time,
	role: "admin" | "user",
};
```

If login is invalid then we return `401 Unauthorized` HTTP code.
# Account creation

The account creation tab is only shown if and only if the user's role is an admin. This requires the admin to enter an email, a password, a name, and a role (which should be a toggle). send to `POST /admin/createAccount`:

```ts
type AccountCreate = {
	email: string,
	password: string,
	name: string,
	role: "admin" | "user",
};
```

If the account is not admin, send `401 Unauthorized`.

If an account does not exist with the following email, the server (or the DB) must generate the corresponding `id`, `password_hashed`,  `account_create_time` (UTC), then attempt to store it. Then it sends `200`.

If an account already exists then we return `409 Conflict` HTTP code.

If a server error occurs, then we return `500` HTTP code.

# Account deletion

The account creation tab is only shown if and only if the user's role is an admin. This requires the admin to enter either an email or the account ID:

```ts
type AccountCreate = { email: string } | { id: string };
```

If the account is not admin, send `401 Unauthorized`.

If the account exists, delete it and send `200`.

If the account does not exist, return `404 Not found`.

# Session, cookie

A session, in the database, must have the shape

```ts
type Session = {
	session_id: byte[128],
	user_id: string,
	start_time: Time,
	ends_time: Time,
	is_valid: bool,
}
```

where `ends_time` = `start_time + 30 days`.

A cookie must contain session_id. All requests needing authorization must have a valid cookie.

At account deletion, all sessions referencing that account must be deleted.