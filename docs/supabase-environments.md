# Supabase environments

Tribe uses a separate Supabase project for each environment:

| Repository environment | Purpose | Supabase project |
| --- | --- | --- |
| Local | Developer machines and disposable test data | Local Supabase CLI stack, or a developer-owned cloud project | 
| Staging | Shared integration testing | Dedicated staging project | 
| Beta | Private TestFlight and API deployment | Dedicated production/beta project | 

Do not point a local or staging client at the beta project. Projects must not
share database passwords, secret API keys, storage buckets, or Auth users.

## Configuration contract

Copy the templates before configuring an environment:

```sh
cp api/.env.example api/.env
cp mobile/.env.example mobile/.env
```

The templates intentionally contain placeholders only. `.env` files are ignored
by Git; do not put their values in source, issue comments, screenshots, or
build logs.

### Mobile client: public configuration

Set these values in `mobile/.env` and in the corresponding Expo build
environment. Every `EXPO_PUBLIC_` value is embedded in the app bundle, so it
must be safe for anyone who installs the app to read.

| Variable | Source | Purpose |
| --- | --- | --- |
| `EXPO_PUBLIC_API_URL` | Deployed API base URL, or local API URL | Tribe API endpoint used by the app. |
| `EXPO_PUBLIC_SUPABASE_URL` | Supabase project's Connect panel | Auth and other Supabase client endpoint. |
| `EXPO_PUBLIC_SUPABASE_PUBLISHABLE_KEY` | Supabase project's Connect panel | Public client key, constrained by Supabase Auth and RLS policies. |

Use the current **publishable** key (`sb_publishable_...`), not a secret or
service-role key. The publishable key is public configuration, but it is not a
substitute for Row Level Security or server-side authorization.

### API: private server configuration

Set these values in `api/.env` for local development and in the API host's
secret manager for staging and beta. Never provide these variables to the Expo
app or prefix them with `EXPO_PUBLIC_`.

| Variable | Required when | Source | Purpose |
| --- | --- | --- | --- |
| `PORT` | Always | Deployment configuration; defaults to `8080` locally | HTTP listener port. |
| `DATABASE_URL` | Database access is implemented | Supabase database connection settings | PostgreSQL connection string, including a password. |
| `SUPABASE_URL` | Supabase Auth or service APIs are implemented | Supabase project's Connect panel | Project base URL. This URL is public, but it remains server configuration here. |
| `SUPABASE_JWT_ISSUER` | JWT verification is implemented | `${SUPABASE_URL}/auth/v1` | Expected `iss` claim. |
| `SUPABASE_JWT_AUDIENCE` | JWT verification is implemented | `authenticated` unless the Auth configuration changes | Expected `aud` claim. |
| `SUPABASE_SECRET_KEY` | A server-only Supabase API call needs elevated access | Supabase project's Connect panel | Secret key; bypasses RLS and must stay in server secret storage. |

The Go API must verify user JWT signatures, issuer, audience, and expiry. With
asymmetric signing keys it should use the project's public JWKS endpoint;
`SUPABASE_SECRET_KEY` is not a JWT-verification secret and must not be used to
authenticate requests.

## Local setup

1. Start a local Supabase stack with the Supabase CLI, or create a
   developer-owned Supabase project. Do not use staging or beta credentials.
2. Get the project URL and publishable key from the CLI output or the project's
   Connect panel. Put them in `mobile/.env`.
3. Get the database connection string and, only if server-side elevated access
   is required, the secret key. Put them in `api/.env`.
4. Set `SUPABASE_JWT_ISSUER` to `${SUPABASE_URL}/auth/v1` and keep
   `SUPABASE_JWT_AUDIENCE=authenticated` unless the project deliberately uses
   a different audience.
5. Start the API with `make api-run` and the mobile app with `make mobile-start`.

The current bootstrap API only requires `PORT`; the remaining API variables are
the agreed contract for the forthcoming Supabase integration. This lets local
development begin without inventing names or exposing credentials.

## Database migrations

Supabase CLI owns the repository's versioned PostgreSQL migrations. The SQL
files in `supabase/migrations/` are applied in filename order. Do not edit a
migration after it has been applied to a shared environment; add a later
migration instead.

`public.users.id` is both the application user identifier and a foreign key to
`auth.users.id`. This direct UUID mapping lets the Go API derive the application
user from a verified Supabase Auth identity without trusting a client-provided
user ID. `avatar_path` stores a private Supabase Storage object path rather
than a permanent public URL; the API will produce authorized, short-lived
signed URLs when avatars are served.

The private `avatars` bucket accepts JPEG, PNG, and WebP files up to 5 MB. The
API stores objects under `<authenticated-user-id>/<random-id>.<extension>` and
exposes `PUT /me/avatar` as a multipart request with an `avatar` file field.
Clients never choose the storage key or persist an avatar association directly;
profile responses contain a short-lived `avatar_url` instead of `avatar_path`.

### Apply locally

With Docker running and the Supabase CLI installed, recreate the local database
and apply every committed migration from a clean state:

```sh
supabase start
supabase db reset
supabase migration list
```

`supabase db reset` is destructive and must only be run against a local
development database.

### Apply to staging or beta

Never run `supabase db reset` against a shared environment. After reviewing a
migration, authenticate and link the CLI to the intended project, then apply
only pending migrations:

```sh
supabase login
supabase link --project-ref <project-ref>
supabase db push
supabase migration list
```

`supabase db push` compares the committed migration files with Supabase's
migration history and applies only missing files in order. The username
migration must be deployed before any application profile rows exist because it
adds a required, unique value.

## Staging and beta setup

1. Create separate Supabase projects and separate API/Expo environment values
   for staging and beta.
2. Configure the API host with the API variables above through its secret
   manager. Do not commit staging or beta `.env` files.
3. Configure Expo/EAS with only the three `EXPO_PUBLIC_` mobile variables for
   the matching environment. Verify the API URL, Supabase URL, and publishable
   key all refer to the same environment before building.
4. Restrict access to database passwords and `SUPABASE_SECRET_KEY` to the API
   deployment identity and the minimum number of maintainers. Rotate either if
   it is exposed.

`SUPABASE_SECRET_KEY`, legacy `service_role` keys, database passwords, and JWT
signing secrets are never mobile configuration. A client must send the user's
Supabase access token to the Go API; the API derives identity from the verified
token rather than from a client-supplied user ID.
