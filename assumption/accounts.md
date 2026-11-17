# `/accounts`

- **Only shown and reachable for `role === 'admin'`**: the nav link is hidden and the route redirects to `/tracking`.
- **Page = two cards**: *Create Account* and *Delete Account*. Listing accounts was not requested, and there
  is no endpoint for it, so there is no list.
- **Create:** name, email, password, confirm password, and a role toggle (User/Admin, default User).
  - **Password rule in the client:** at least 8 characters. OWASP's minimum is 8 with MFA and 15 without; the
    requirements only fix the hashing (Argon2id, 19 MiB, t=2, p=1), which is server-side. The confirm field
    is a UI-only safeguard. The final rule should come from the server.
  - Name and email are trimmed before sending.
  - Errors: 409 "already exists", 401 "not allowed", 500 "server error, not created", and anything else shows
    the server's message.
  - On success the form clears and shows the new account's id (assumes the response contains `{ id, name }`).
- **Delete:** toggle *By email* / *By account ID*, then a **two-step confirm** (the button turns red and asks
  again), because deletion cannot be undone.
  - 404 "No account found", 401 "not allowed".
  - **Deleting your own account:** the spec deletes all of that account's sessions, so after a successful
    delete the client re-checks `/me` and goes to `/login` if the session is gone.
  - Cached creator-name lookups are invalidated after a delete.
- The spec's deletion type is named `AccountCreate` (copy-paste). The client calls it `AccountDelete`.
