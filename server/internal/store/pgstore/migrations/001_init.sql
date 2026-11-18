-- Initial schema. Times are timestamptz holding UTC values at millisecond precision.
-- Ids are text (UUIDv7 strings) so that malformed ids in requests are simply "not found"
-- instead of cast errors.

CREATE TABLE accounts (
    id                  text        PRIMARY KEY,
    email               text        NOT NULL,
    password_hashed     text        NOT NULL,
    name                text        NOT NULL,
    account_create_time timestamptz NOT NULL,
    role                text        NOT NULL CHECK (role IN ('admin', 'user'))
);
-- Emails are unique case-insensitively.
CREATE UNIQUE INDEX accounts_email_lower_key ON accounts (lower(email));

CREATE TABLE sessions (
    session_id bytea       PRIMARY KEY CHECK (octet_length(session_id) = 16),
    -- Deleting an account deletes all of its sessions.
    user_id    text        NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    start_time timestamptz NOT NULL,
    ends_time  timestamptz NOT NULL,
    is_valid   boolean     NOT NULL
);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);

CREATE TABLE trackings (
    tracking_number text        PRIMARY KEY,
    cont            text,
    seal            text,
    pol             text        NOT NULL,
    pod             text        NOT NULL,
    -- physicalDetails: either all of weight + dimensions are set, or none (pcs_count is optional).
    pcs_count       integer,
    weight_kg       double precision,
    length_cm       double precision,
    width_cm        double precision,
    height_cm       double precision,
    -- No foreign key: trackings outlive the account that created them.
    creator_id      text        NOT NULL,
    creation_time   timestamptz NOT NULL,
    customer_id     text,

    -- Case/accent-folded copies for filtering and sorting (computed by the app, see textnorm).
    tracking_number_fold text NOT NULL,
    pol_fold             text NOT NULL,
    pod_fold             text NOT NULL,

    -- Denormalised last status, maintained in the same transaction as every status insert.
    last_status_id              text,
    last_status_occurring_time  timestamptz,
    last_status_creation_time   timestamptz,

    CHECK ((weight_kg IS NULL) = (length_cm IS NULL)
       AND (weight_kg IS NULL) = (width_cm IS NULL)
       AND (weight_kg IS NULL) = (height_cm IS NULL)),
    CHECK (pcs_count IS NULL OR weight_kg IS NOT NULL)
);
CREATE INDEX trackings_last_status_idx ON trackings (last_status_occurring_time DESC, last_status_creation_time DESC);
CREATE INDEX trackings_creation_time_idx ON trackings (creation_time);

CREATE TABLE statuses (
    id              text        PRIMARY KEY,
    tracking_number text        NOT NULL REFERENCES trackings (tracking_number) ON DELETE CASCADE,
    status_type     text        NOT NULL CHECK (status_type IN ('good', 'bad')),
    status          text        NOT NULL CHECK (status <> ''),
    -- Good statuses always have an array (possibly empty); bad statuses have NULL.
    warnings        text[],
    occurring_time  timestamptz NOT NULL,
    location        text        NOT NULL,
    creator_id      text        NOT NULL,
    creation_time   timestamptz NOT NULL,
    CHECK ((status_type = 'good') = (warnings IS NOT NULL))
);
CREATE INDEX statuses_order_idx ON statuses (tracking_number, occurring_time, creation_time, id);

ALTER TABLE trackings
    ADD CONSTRAINT trackings_last_status_fk
    FOREIGN KEY (last_status_id) REFERENCES statuses (id) DEFERRABLE INITIALLY DEFERRED;
