CREATE TABLE users
(
    id           UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    username     STRING(50)  NOT NULL UNIQUE,
    email        STRING(254) NOT NULL UNIQUE,
    password     STRING      NOT NULL,
    is_superuser BOOL                 DEFAULT false,
    last_login   TIMESTAMPTZ NULL,
    created_at   TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL,
    deleted_at   TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01',
    UNIQUE (username, deleted_at),
    UNIQUE (email, deleted_at)
);
