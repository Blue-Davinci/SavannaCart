-- name: CreateNewUser :one
INSERT INTO users (
    first_name,
    last_name,
    email,
    profile_avatar_url,
    password,
    oidc_sub,
    role_level,
    activated
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
 RETURNING id, role_level, version, created_at, updated_at, last_login;

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