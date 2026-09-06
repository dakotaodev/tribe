
### `AGENTS.md`

```markdown
# Codex Instructions — Tribe

## Source of Truth

Before implementing significant changes, read:

1. `docs/product.md`
2. `docs/mvp.md`
3. `docs/architecture.md`

Those files define the current product and technical constraints.

If existing code conflicts with these documents, call out the discrepancy before silently changing product behavior.

## Product Context

Tribe is a private social network built around finite reciprocal relationships.

The central product constraint is:

**A maximum of 150 accepted reciprocal connections per user.**

The application should strengthen relationships rather than maximize engagement.

## Hard Product Constraints

Unless explicitly instructed otherwise, do not introduce:

- follower/following mechanics
- more than 150 accepted connections
- algorithmic feed ranking
- recommended posts
- public engagement metrics
- trending content
- reposts
- direct messages
- video
- stories
- public discovery feeds
- creator/influencer features
- advertising
- AI features

Do not reinterpret a technical task as permission to expand product scope.

## Working Style

For non-trivial tasks:

1. Read the relevant documentation.
2. Inspect the existing implementation.
3. Identify existing patterns that should be preserved.
4. Propose a concise implementation plan.
5. Identify schema, authorization, privacy, or concurrency concerns.
6. Implement only the requested scope.
7. Add or update relevant tests.
8. Run project verification commands.
9. Summarize what changed.
10. Explicitly state anything that could not be verified.

Do not make unrelated refactors while implementing a feature.

## Scope Discipline

Prefer small vertical changes over large speculative implementations.

A task should generally correspond to one GitHub issue or one clearly defined outcome.

If a request contains several separable features, recommend splitting the work before implementing unless they are tightly coupled.

## Backend

Backend technology:

- Go
- Gin for the HTTP API
- REST
- PostgreSQL
- modular monolith

Gin is the standard HTTP framework. Use it for routing, middleware, request binding, and HTTP responses; do not introduce `net/http` routers or alternative frameworks.

Prefer simple, explicit Go.

Avoid unnecessary abstraction.

Gin handlers should remain thin. They may bind/validate transport input, read authenticated identity from context, call application/domain logic, and serialize HTTP responses.

Business rules must not be embedded directly in handlers. Application/domain code must remain framework-independent: do not pass `gin.Context`, Gin request types, or Gin response helpers beyond the transport layer; use plain Go inputs and `context.Context` instead.

Database logic should have clear boundaries.

Errors must be handled explicitly.

## Mobile

Mobile technology:

- React Native
- Expo
- TypeScript
- iOS first

Prefer established iOS interaction patterns.

UI should feel:

- calm
- minimal
- warm
- private
- human

Do not add visual gamification merely to increase engagement.

Reuse existing components and design tokens.

Do not introduce new colors, fonts, spacing systems, or component libraries casually.

## Authentication

Supabase Auth provides authentication.

The Go API must verify authenticated identity.

Never trust:

- client-provided user IDs
- hidden UI elements
- route parameters

as authorization.

## Authorization

Authorization rules belong on the server.

Always consider:

- ownership
- relationship status
- blocks
- privacy
- requested resource visibility

For security-sensitive changes, add negative tests.

Examples:

- stranger cannot retrieve private post
- user cannot delete another user's post
- requester cannot accept their own outgoing request
- blocked user cannot access prohibited content

## Database Changes

Treat migrations as high-risk changes.

Before modifying schema:

1. inspect existing migrations
2. explain the proposed change
3. consider uniqueness constraints
4. consider indexes
5. consider rollback/data implications

Do not destructively modify shared data without explicit approval.

## 150-Person Invariant

The connection limit is a backend invariant.

Any code that creates an accepted connection must preserve:

```text
connection_count(user) <= 150
```text
connection_count(user) <= 150
for both participants.
Consider concurrency explicitly.
Tests should include cases such as:
149 → accept → succeeds
150 → accept → fails
other participant at 150 → fails
duplicate relationship → fails
self relationship → fails
concurrent acceptance → never produces 151
Feed
The MVP feed is reverse chronological.
Do not introduce ranking.
Do not add recommendation systems.
Feed authorization must be tested.
Testing
Backend features should include tests for important business behavior.
Before completing backend work, run:
go test ./...
go vet ./...
Ensure Go code is formatted.
Before completing mobile work, run the project's configured TypeScript/typecheck command.
When available, prefer the root verification command:
make test
Code Review Priorities
Review changes in roughly this order:
1. security
2. privacy
3. data integrity
4. authorization
5. concurrency
6. business invariants
7. API behavior
8. tests
9. maintainability
10. styling
Dependencies
Prefer existing dependencies.
Before adding a new production dependency:
- explain why it is useful
- check whether the standard library or existing dependency already solves the problem
- avoid large dependencies for trivial behavior
Completion Report
After implementing a task, provide:
Changed
A concise description of behavior implemented.
Files
Important files added or modified.
Verification
Commands/tests executed and their results.
Risks
Any unresolved concern or behavior not verified.
Out of Scope
Anything intentionally not implemented.
Product Decisions
Codex may make local implementation decisions.
Codex should not independently make major product decisions.
If implementation requires choosing between materially different product behaviors, surface the decision rather than silently inventing product requirements.