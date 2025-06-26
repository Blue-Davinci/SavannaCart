-- name: CreateNewUser :one
INSERT INTO users (
    first_name,
    last_name,
    email,
    profile_avatar_url,
    phone_number,
    password,
    oidc_sub
) VALUES ($1, $2, $3, $4, $5, $6, $7)
 RETURNING id, role_level,activated, version, created_at, updated_at, last_login;

-- name: GetUserByID :one
SELECT
    id,
    first_name,
    last_name,
    email,
    profile_avatar_url,
    phone_number,
    password,
    oidc_sub,
    role_level,
    activated,
    version,
    created_at,
    updated_at,
    last_login
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT
    id,
    first_name,
    last_name,
    email,
    profile_avatar_url,
    phone_number,
    password,
    oidc_sub,
    role_level,
    activated,
    version,
    created_at,
    updated_at,
    last_login
FROM users
WHERE email = $1;

-- name: UpdateUser :one
UPDATE users
SET
    first_name = $1,
    last_name = $2,
    email = $3,
    profile_avatar_url = $4,
    phone_number = $5,
    password = $6,
    role_level = $7,
    activated = $8,
    version = version + 1,
    updated_at = NOW(),
    last_login = $9
WHERE id = $10 AND version = $11
RETURNING updated_at, version;