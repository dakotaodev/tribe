# Tribe

Tribe is a private, reciprocal social network with a maximum of 150 accepted
connections per person. This repository contains its Go API and Expo mobile
client.

## Prerequisites

- Go 1.25 or newer
- Node.js 20 or newer
- pnpm 9 or newer

## Getting started

1. Follow the [Supabase environment setup](docs/supabase-environments.md) to
   create or select the correct local project, then copy the environment
   templates and fill in that project's values:

   ```sh
   cp api/.env.example api/.env
   cp mobile/.env.example mobile/.env
   ```

2. Install mobile dependencies:

   ```sh
   cd mobile
   pnpm install
   ```

3. In separate terminals, start the API and mobile app:

   ```sh
   make api-run
   make mobile-start
   ```

The API listens on `http://localhost:8080` by default. Its health endpoint is
available at `GET /health`.

The shared authenticated current-user profile boundary is defined in the
[auth and profile contract](docs/auth-profile-contract.md). Backend and mobile
work should implement that contract without changing it independently.

## Staging deployment

Deploy the shared Render staging API using the
[Render staging deployment guide](docs/render-staging.md).

## Verification

Run all currently configured checks from the repository root:

```sh
make test
```

For release builds, use Expo Application Services after configuring the project
for the intended environment. Product features beyond the initial schema have
intentionally not been implemented yet.
