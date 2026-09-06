# Tribe — Architecture

## Goals

The initial architecture should optimize for:

1. development speed
2. correctness
3. privacy
4. simple deployment
5. ease of modification
6. sufficient reliability for a private beta

It should not optimize prematurely for massive scale.

## System Overview

```text
┌──────────────────────────┐
│ Expo / React Native      │
│ iOS-first mobile client  │
└─────────────┬────────────┘
              │ HTTPS / JSON
              ▼
┌──────────────────────────┐
│ Go REST API              │
│ Modular monolith         │
└─────────────┬────────────┘
              │
        ┌─────┴──────────────┐
        ▼                    ▼
┌────────────────┐   ┌────────────────┐
│ Supabase       │   │ Supabase       │
│ PostgreSQL     │   │ Storage        │
└────────────────┘   └────────────────┘

Authentication:
Expo ↔ Supabase Auth
         ↓ token
Expo → Go API
         ↓
Go validates Supabase JWT

Mobile
Technology:
- React Native
- Expo
- TypeScript
- iOS first
Android support is intentionally deferred, but implementation should avoid unnecessary iOS-only application logic where Expo already provides portable abstractions.
Backend
Technology:
- Go
- Gin for the HTTP API
- REST
- single deployable API service

Gin is the standard HTTP framework for this service. Use it for routing, request binding, middleware, request-scoped values, and JSON responses.

Prefer:
- Gin at the transport boundary
- small focused dependencies
- explicit behavior
- straightforward code

Avoid:
- alternative HTTP frameworks or routers
- generic repository frameworks
- dependency injection frameworks
- premature abstractions
- distributed architecture
Backend Package Structure
Prefer domain-oriented packages.
Example:
api/
├── cmd/
│   └── server/
│       └── main.go
│
├── internal/
│   ├── auth/
│   ├── users/
│   ├── connections/
│   ├── posts/
│   ├── comments/
│   ├── feed/
│   ├── moderation/
│   └── analytics/
│
├── migrations/
├── go.mod
└── Dockerfile
Avoid organizing the entire application exclusively as:
controllers/
services/
repositories/
models/
Domain boundaries should remain visible.
HTTP Handlers
Gin handlers should primarily:
1. bind and validate basic request input
2. obtain authenticated identity from Gin middleware/context
3. translate HTTP input into framework-independent application inputs
4. invoke domain/application logic
5. serialize the result or error as an HTTP response

Handlers must remain thin. Business rules must not live directly inside handlers, and domain/application packages must not depend on `gin.Context`, Gin request types, or Gin response helpers. Pass plain Go values and `context.Context` across the transport boundary.
Authentication
Supabase Auth is the identity provider.
The mobile client receives an authentication token from Supabase.
The Go API:
1. receives the bearer token
2. verifies the token
3. derives authenticated identity from verified claims
4. maps the external auth identity to the internal user record
Never trust a user ID supplied by the client as proof of identity.
Authorization
Authorization is enforced server-side.
Examples:
- users may delete only their own posts
- users may modify only their own profile
- users may see private posts only when permitted
- connection requests may only be accepted by the intended recipient
The mobile UI may hide inaccessible actions, but UI state is not an authorization control.
Database
Primary database:
PostgreSQL through Supabase
Initial entities are expected to include:
users
connection_requests
connections
posts
post_media
comments
reactions
blocks
reports
Exact schema may evolve during implementation.
Identifiers
Prefer UUIDs for application entities unless a strong reason exists otherwise.
Do not expose sequential identifiers unnecessarily.
Migrations
All schema changes must be represented by committed database migrations.
Do not manually modify production schema without corresponding migration files.
Migrations should be reviewed carefully before execution against shared environments.
Connections
Connections are reciprocal.
The schema should make it difficult or impossible to accidentally represent:
A connected to B
B not connected to A
If the chosen representation stores two IDs in a single connection record, establish canonical ordering or equivalent uniqueness constraints.
Duplicate accepted relationships must not be possible.
150 Connection Invariant
The maximum accepted connection count is:
150
This must be enforced by the backend.
The implementation must remain correct under concurrent acceptance attempts.
Example failure to prevent:
current count: 149

request A accepted concurrently
request B accepted concurrently

result: 151
The implementation should use appropriate transaction and locking semantics.
The client must not be responsible for enforcing this rule.
Feed
The MVP feed is:
- connections-only
- reverse chronological
- cursor paginated
Conceptually:
posts where author is:
    authenticated user
    OR accepted connection

ORDER BY created_at DESC
Blocked relationships must be respected.
Do not introduce:
- engagement scoring
- ranking models
- personalization
- recommendations
Pagination
Prefer cursor pagination for feeds.
Avoid offset pagination for unbounded timelines.
A cursor may initially use an ordered combination such as:
(created_at, id)
to provide deterministic ordering.
Media
Media storage:
Supabase Storage
The MVP supports images only.
Maximum:
4 images per post
Do not proxy large image bodies unnecessarily through the Go API if signed/direct upload flows provide a simpler and secure architecture.
The backend must still control authorization over media ownership and attachment to posts.
Security
Treat the following areas as high-review changes:
- authentication
- authorization
- database migrations
- privacy rules
- connection invariants
- account deletion
- blocking
- media access
- secrets
Never commit:
- API secrets
- private service keys
- production credentials
- signing secrets
Use environment variables and .env.example files.
Deployment
Initial target:
API
Render or similarly simple managed container/web-service hosting.
Database/Auth/Storage
Supabase.
Mobile
Expo EAS Build.
Private iOS distribution:
TestFlight.
Environments
Initially support:
- local development
- shared development/staging
- production/private beta
Do not build elaborate environment-management infrastructure.
Observability
Initial requirements:
- structured backend logs
- request/error logging
- basic health endpoint
Once external TestFlight testing begins, add an error-reporting product such as Sentry if useful.
Do not build a custom observability platform.
Testing
Backend business rules should have automated tests.
High-priority areas include:
- authorization
- connection requests
- connection acceptance
- 150 limit
- concurrent acceptance
- post privacy
- blocks
- deletion ownership
Mobile testing may initially prioritize:
- TypeScript correctness
- component/unit tests for significant logic
- manual end-to-end testing of critical flows
CI
At minimum, CI should run:
go test ./...
go vet ./...
gofmt verification
TypeScript typecheck
Additional checks may be added only when useful.
Architecture Non-Goals
Do not introduce without demonstrated need:
- microservices
- Kubernetes
- Kafka
- Redis
- GraphQL
- event sourcing
- CQRS
- custom authentication
- custom object storage
- custom recommendation infrastructure
- separate feed service
- service mesh