# Auth and Current-User Profile Contract

This document is the contract between the Go API and Expo client for
authentication and the caller's profile. It applies to the authenticated
current-user endpoints only; it does not define public profile discovery.

Implementations of either client or server must not change this contract
without coordinating the change first.

## Authentication and identity

Every request to an endpoint in this document must send a Supabase access token:

```http
Authorization: Bearer <supabase-access-token>
```

The API verifies the JWT before handling the request and derives the caller's
application user ID exclusively from the verified `sub` claim. The API maps it
directly to `public.users.id`, which is a foreign key to `auth.users.id`.

There is no `user_id` request field or route parameter for these endpoints.
If one is supplied as an unknown JSON field, the request is invalid. A client
must never be able to read or modify a different user's profile through these
endpoints.

The API returns `401` for a missing, malformed, expired, or otherwise invalid
bearer token.

## Profile representation

The profile returned by this API has the following shape:

```json
{
  "id": "f83f6ae5-0c18-4b1f-91ef-f7584793af9b",
  "username": "dakota",
  "display_name": "Dakota",
  "bio": "Building a quieter social network.",
  "avatar_path": "avatars/f83f6ae5-0c18-4b1f-91ef-f7584793af9b/avatar.jpg",
  "created_at": "2026-09-06T19:00:00Z",
  "updated_at": "2026-09-06T19:02:00Z"
}
```

`id`, `created_at`, and `updated_at` are server-managed and read-only.
`avatar_path` is either a private Storage object path or `null`; it is not a
public URL. Signed avatar URLs, if needed, are a separate future contract.

### Writable fields and validation

`PUT /me` accepts only the following JSON fields:

| Field | Required | Rules |
| --- | --- | --- |
| `username` | Yes | String of 3–30 characters; unique across users. |
| `display_name` | Yes | String of 1–80 characters. |
| `bio` | No | String of 0–500 characters; omitted means `""` when creating and preserves the existing value when updating. |
| `avatar_path` | No | String private Storage object path or `null`; omitted preserves the existing value when updating and is `null` when creating. |

Character counts use PostgreSQL `char_length`, matching the database schema.
The API treats username uniqueness as case-sensitive until a later migration
and contract explicitly change that behavior.

## Profile lifecycle

Supabase account creation and application profile creation are separate steps:

1. A person authenticates with Supabase.
2. The API verifies the token and looks for `public.users.id = sub`.
3. If no row exists, the person needs onboarding.
4. `PUT /me` with the required profile fields creates that caller's row.
5. Later `PUT /me` requests update that same row.

The API never automatically creates a profile during `GET /me`. This lets the
mobile client reliably choose between profile setup and the authenticated app.

## Endpoints

### `GET /me`

Returns the authenticated caller's profile.

**Success — `200 OK`**

```json
{
  "data": {
    "id": "f83f6ae5-0c18-4b1f-91ef-f7584793af9b",
    "username": "dakota",
    "display_name": "Dakota",
    "bio": "Building a quieter social network.",
    "avatar_path": null,
    "created_at": "2026-09-06T19:00:00Z",
    "updated_at": "2026-09-06T19:00:00Z"
  }
}
```

**Onboarding needed — `404 Not Found`**

```json
{
  "error": {
    "code": "PROFILE_ONBOARDING_REQUIRED",
    "message": "Create your profile to continue."
  }
}
```

This is an expected authenticated state, not an invalid token or a request for
another user's profile.

### `PUT /me`

Creates the caller's profile when absent, otherwise updates the caller's
profile. The same request shape is used for both cases.

```http
PUT /me
Content-Type: application/json
Authorization: Bearer <supabase-access-token>

{
  "username": "dakota",
  "display_name": "Dakota",
  "bio": "Building a quieter social network.",
  "avatar_path": null
}
```

**Created — `201 Created`** and **updated — `200 OK`** both return the current
profile in the standard success envelope:

```json
{
  "data": {
    "id": "f83f6ae5-0c18-4b1f-91ef-f7584793af9b",
    "username": "dakota",
    "display_name": "Dakota",
    "bio": "Building a quieter social network.",
    "avatar_path": null,
    "created_at": "2026-09-06T19:00:00Z",
    "updated_at": "2026-09-06T19:02:00Z"
  }
}
```

## Error envelope

All errors use this envelope:

```json
{
  "error": {
    "code": "MACHINE_READABLE_CODE",
    "message": "Human-readable explanation.",
    "fields": {
      "field_name": "Validation explanation."
    }
  }
}
```

`fields` is omitted unless validation identifies one or more request fields.
Clients should branch on `error.code`, not the message text.

| Status | Code | When it is returned |
| --- | --- | --- |
| `400` | `INVALID_REQUEST` | The JSON body is malformed, has unknown fields, or a required field is absent. |
| `401` | `UNAUTHENTICATED` | The bearer token is missing or cannot be verified. |
| `404` | `PROFILE_ONBOARDING_REQUIRED` | `GET /me` found no application profile for an otherwise authenticated caller. |
| `409` | `USERNAME_TAKEN` | The requested username is already owned by another user. |
| `422` | `VALIDATION_FAILED` | A writable field violates its documented type or length rules. |
| `500` | `INTERNAL_ERROR` | An unexpected server failure occurred. |

For example, a username conflict is:

```json
{
  "error": {
    "code": "USERNAME_TAKEN",
    "message": "That username is unavailable.",
    "fields": {
      "username": "Choose a different username."
    }
  }
}
```

## Implementation boundaries

Gin is responsible only for HTTP binding, authentication context lookup, and
response serialization. Profile lifecycle, validation, uniqueness handling,
and database access belong in framework-independent application/domain code.

The profile write must use the verified caller identity, including during an
upsert or conflict retry. Database uniqueness remains the final authority for
username conflicts; the API translates that specific conflict to
`USERNAME_TAKEN`.
