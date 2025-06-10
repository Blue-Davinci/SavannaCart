-- name: CreateNewUser :one
INSERT INTO users (
    first_name,
    last_name,
    email,
    profile_avatar_url,
    password,
    oidc_sub
) VALUES ($1, $2, $3, $4, $5, $6)
 RETURNING id, role_level,activated, version, created_at, updated_at, last_login;

-- name: GetUserByEmail :one
SELECT
    id,
    first_name,
    last_name,
    email,
    profile_avatar_url,
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
    password = $5,
    role_level = $6,
    activated = $7,
    version = version + 1,
    updated_at = NOW(),
    last_login = $8
WHERE id = $9 AND version = $10
RETURNING updated_at, version;