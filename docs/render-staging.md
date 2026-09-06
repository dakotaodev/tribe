# Render staging deployment

This runbook deploys the Tribe Go API as the shared staging service. It uses
Render for the API and a dedicated Supabase staging project for future
database, Auth, and Storage integrations. It does not create a production
service or deploy the Expo app.

## Prerequisites

- Access to the `dakotaodev/tribe` GitHub repository and its `develop` branch.
- A Render account with permission to create services.
- A dedicated Supabase staging project. Do not use local or beta credentials.

## Create the staging branch

Create the branch once from the current integration baseline, then push it to
GitHub:

```sh
git switch -c develop main
git push -u origin develop
```

Subsequent commits pushed to `develop` automatically deploy the staging API.

## Create the Render service

1. In Render, select **New** > **Blueprint** and connect the
   `dakotaodev/tribe` repository.
2. Select the `develop` branch and use the root `render.yaml` Blueprint. It
   creates the `tribe-api-staging` public Docker web service in Oregon.
3. During the initial Blueprint setup, enter values from the dedicated Supabase
   staging project for the prompted variables:

   | Variable | Source |
   | --- | --- |
   | `DATABASE_URL` | Supabase database connection settings |
   | `SUPABASE_URL` | Supabase Connect panel |
   | `SUPABASE_JWT_ISSUER` | `${SUPABASE_URL}/auth/v1` |

   These are configured with `sync: false`: their values stay in Render and
   are never stored in the repository. `DATABASE_URL` contains a password and
   must be treated as a secret. The URL and issuer are not public mobile
   configuration, even though they are not credentials.
4. Start the deploy and use the Render logs to confirm that the API starts and
   binds to the `PORT` supplied by Render. Do not add a fixed `PORT` value in
   Render; the API uses `8080` only for local development.

The Blueprint uses `api/Dockerfile`, `api/` as the Docker build context, and
`/health` as the health-check path. Render will keep the public
`onrender.com` hostname enabled; no custom domain is required for staging.

## Verify the deployment

After Render reports the deployment healthy, replace the placeholder below
with the service's HTTPS hostname:

```sh
curl --fail --silent --show-error https://<render-service-host>/health
```

The response must be:

```json
{"status":"ok"}
```

Push a harmless commit to `develop` and confirm Render begins a new automatic
deploy. Render cancels a new deployment that cannot pass the HTTP health
check, leaving the previously healthy version serving traffic.

## Configure the staging mobile build

Set the Render HTTPS base URL as the staging Expo/EAS value:

```text
EXPO_PUBLIC_API_URL=https://<render-service-host>
```

`EXPO_PUBLIC_API_URL` is embedded in the mobile app and is intentionally
public. Configure it alongside the matching staging Supabase URL and
publishable key as described in [Supabase environments](supabase-environments.md).
Never add a database password, Supabase secret key, or other server credential
to Expo/EAS public variables.

## Ongoing operations

- Rotate Supabase credentials in Supabase and update the corresponding Render
  environment values; redeploy after each rotation.
- Do not put secrets in `render.yaml`, `.env` files committed to Git, issue
  comments, screenshots, or build logs.
- Change the Render region, plan, branch, or health path through a reviewed
  Blueprint change. The initial free plan is intended only for shared staging.
