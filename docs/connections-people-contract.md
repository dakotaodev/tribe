# Reciprocal Connections and People Contract

Status: **Proposed for review**, not yet locked in Linear.

Issue: [TRI-7 — Contract: reciprocal connections and People](https://linear.app/tribe-workspace/issue/TRI-7/contract-reciprocal-connections-and-people).

This is the proposed shared boundary for TRI-8 through TRI-13. Backend owns
migrations, Gin transport, framework-independent domain logic, and tests. Mobile
owns Expo screens, client mapping, mock states, and UI tests. Once agreed, record
the contract in TRI-7 before implementation begins; subsequent contract changes
must update that issue first.

## Outcome and scope

A person can send a request to a known user, see incoming requests, explicitly
accept or reject one, see their accepted connections and their own `x / 150`
count, and remove a connection. Acceptance establishes one reciprocal
relationship. Neither participant may exceed 150 accepted connections.

This follows [product](docs-product.md), [MVP](docs-mvp.md),
[architecture](architecture.md), and [auth/profile](auth-profile-contract.md).
The product/MVP filenames differ from the shorter names referenced in AGENTS.MD.
Existing migrations contain users but no social graph tables. This document
proposes schema without applying it.

## Product choices proposed for agreement

1. **Crossing requests require explicit acceptance.** If A has requested B,
   B's attempt to request A returns the existing incoming-request conflict.
   Sending never automatically accepts a request.
2. **Only accepted relationships consume capacity.** Requests remain possible
   at 150. Acceptance checks both users atomically; failure leaves the request
   pending.
3. **Rejection and removal allow a later fresh request.** They do not
   permanently prohibit contact. Blocking is a separate safety policy.
   Requests do not expire automatically under this contract.
4. **Address a known recipient by UUID.** No username search, directory,
   contact upload, or public discovery is introduced. Invitation resolution
   and permission to expose identities belong to TRI-28. This API does not
   itself provide a way to obtain another user's UUID.
5. **People shows the caller's count only.** No other user's count, mutual
   connection list, or third-party graph is exposed.

## Identity and privacy

Every endpoint requires verified bearer identity under the auth/profile
contract and an existing application profile. Missing/invalid credentials
return `401 UNAUTHENTICATED`; a valid caller without a profile receives
`404 PROFILE_ONBOARDING_REQUIRED`.

The server derives the caller/sender from verified identity. Recipient UUIDs
and route IDs identify targets, never establish authority. Reject unknown JSON
fields, including caller/sender identity overrides.

Only the recipient may accept or reject. Incoming lists contain only requests
addressed to the caller. Connection lists contain only the caller's
relationships. Either participant may remove their relationship. For decision
and removal endpoints, an absent resource and a resource outside the caller's
authority both return `404 RESOURCE_NOT_FOUND`, without participant details.
A sender cannot accept or reject their own outgoing request.

Pending requests confer no private-post access. Removal revokes
relationship-based access in both directions for subsequent authorized reads;
previously downloaded content cannot be recalled. Feed/media implementations
must check current authorization rather than trust a cached People list.

TRI-32 owns blocking semantics, including persistence, list filtering, and
request cleanup. Reserve `403 CONNECTION_NOT_ALLOWED` for prohibited send or
accept operations; do not identify who blocked whom. Mobile must tolerate this
result. Safety integration must serialize block/accept races and enforce its
final bilateral policy before beta. Blocking is not implemented by this draft.

## Canonical persistence

Proposed tables use UUID primary keys and server-managed UTC timestamps:

| Table | Columns and constraints |
| --- | --- |
| `connection_requests` | `id`, `sender_id`, `recipient_id`, canonical `user_low_id`/`user_high_id`, `status` (`pending`, `accepted`, `rejected`), `created_at`, nullable `resolved_at` |
| `connections` | `id`, `user_low_id`, `user_high_id`, unique `request_id`, `created_at` |

Participants reference `public.users.id`. Canonical order uses PostgreSQL UUID
ordering: low < high. Request low/high values are generated from sender and
recipient or constrained to exactly their minimum/maximum. Self pairs are
invalid. A partial unique index on the request pair where status is `pending`
prevents same-direction and crossing pending duplicates. A unique constraint
on the connection pair prevents duplicate accepted relationships.

`resolved_at` is null exactly when a request is pending. Acceptance marks the
request accepted and inserts its connection in one transaction. The connection
must reference an accepted request with the same participants; domain logic
enforces this alongside database constraints. Retain resolved requests for
retry recognition, with no history endpoint. Removal deletes the connection
and leaves its request resolved. A fresh request receives a new ID.

No pending request may coexist with an accepted connection for a pair. This
cross-table rule requires the serialized mutation protocol below, not merely
the two uniqueness indexes. Resolved historical requests may coexist with
later requests/connections.

Index pending incoming requests on `(recipient_id, created_at DESC, id DESC)`.
Index connections separately on `(user_low_id, created_at DESC, id DESC)` and
`(user_high_id, created_at DESC, id DESC)` for listing and counting.

Follow existing application-table boundaries: enable RLS and revoke direct
`anon`/`authenticated` access. The Go API owns application authorization.
Introduce no client-write policies or broadly callable privileged functions.

Use new migrations without rewriting history. Foreign keys should cascade
graph rows when a participant's application profile is deleted; full deletion
and retention policy remains TRI-32's responsibility. Dropping these tables
loses relationship data, so rollback in a shared environment requires separate
approval. Historical-request retention beyond retry support must be reconciled
with the safety contract before beta.

## Response types

IDs are UUID strings; timestamps are RFC 3339 UTC strings. Required fields are
always present; nullable values are explicit `null`.

```ts
type PersonSummary = {
  id: string;
  username: string;
  display_name: string;
  avatar_path: string | null;
};

type ConnectionRequest = {
  id: string;
  sender: PersonSummary;
  recipient: PersonSummary;
  status: "pending" | "accepted" | "rejected";
  created_at: string;
  resolved_at: string | null;
};

type Connection = {
  id: string;
  person: PersonSummary; // always the other participant
  created_at: string;   // time of acceptance
};

type Page<T> = {
  data: T[];
  page: { next_cursor: string | null; has_more: boolean };
};

type PeoplePage = Page<Connection> & {
  meta: { accepted_count: number; connection_limit: 150 };
};
```

Person summaries contain current profile values, not historical snapshots.
They omit email, bio, auth metadata, and other people's connection counts.
`avatar_path` follows the existing private-path convention; it is not a public
or signed URL. Authorized avatar download is a separate contract; mobile uses
a placeholder until authorized rendering is available.

## Endpoints

JSON success responses use the existing `data` envelope.

| Method and path | Input | Success |
| --- | --- | --- |
| `POST /connection-requests` | `{ "recipient_id": "<uuid>" }` | `201 { "data": ConnectionRequest }` |
| `GET /connection-requests/incoming` | Optional `limit`, `cursor` | `200 Page<ConnectionRequest>`; pending only |
| `POST /connection-requests/:id/accept` | No body | `200 { "data": Connection }` |
| `POST /connection-requests/:id/reject` | No body | `200 { "data": ConnectionRequest }`; status rejected |
| `GET /connections` | Optional `limit`, `cursor` | `200 PeoplePage` |
| `DELETE /connections/:id` | No body | `204`, no response body |

Nonempty bodies on bodyless endpoints are invalid. Empty lists return
`data: []`, `next_cursor: null`, and `has_more: false`. People metadata appears
on every page, including empty pages. No outgoing-list, cancellation, bulk
action, or third-party graph endpoint is included.

### Decisions, conflicts, and retries

| Situation | Required outcome |
| --- | --- |
| Send to self | `422 SELF_CONNECTION_NOT_ALLOWED` |
| Send to nonexistent/unavailable recipient | `404 RESOURCE_NOT_FOUND`, no profile payload |
| Send when already connected | `409 ALREADY_CONNECTED` |
| Repeat send while same outgoing request is pending | `409 REQUEST_ALREADY_PENDING`, existing ID in `details.request_id` |
| Send while an incoming request from that person is pending | `409 INCOMING_REQUEST_EXISTS`, existing ID in `details.request_id`; no auto-accept |
| Accept pending request, both below capacity | Create one connection and resolve the request atomically |
| Accept with either participant at capacity | `409 CONNECTION_LIMIT_REACHED`; no mutation |
| Repeat accept on accepted request with its original connection present | `200` with that connection, even if someone is now at 150; no mutation |
| Accept rejected request, or accepted request whose original connection was removed | `409 REQUEST_ALREADY_RESOLVED`; never recreate a connection |
| Reject pending request | Resolve as rejected; create no connection |
| Repeat reject on rejected request | `200` with rejected request; no mutation |
| Reject accepted request | `409 REQUEST_ALREADY_RESOLVED` |
| Remove existing connection as either participant | Delete the reciprocal relationship once |
| Repeat remove after deletion | `404 RESOURCE_NOT_FOUND`; client may treat as already absent |

Old request IDs cannot affect a later relationship between the same people.
For concurrent accept/reject, one transition wins; the other observes the
committed state and follows this table. A new request after rejection has a
new ID, so retrying an old rejection cannot reject it.

Authorize before returning lifecycle conflicts or retry results. Reserved
safety restrictions take precedence over disclosing prohibited relationships.
For a new send, check self/target, accepted relationship, then pending
relationship. For acceptance, check authorized lifecycle/retry state before
checking capacity.

### Pagination and counts

Both lists use reverse chronological `(created_at DESC, id DESC)` ordering and
keyset pagination. `limit` defaults to 30; valid integers are 1–100. Invalid
limits and malformed/incompatible cursors return `400 INVALID_REQUEST`.

Cursors are opaque to clients, versioned, and scoped to the caller and list.
They carry the last returned ordering key and never grant authorization.
`has_more` is true exactly when another authorized row exists in that read;
only then is `next_cursor` non-null. Recheck authorization on every page.

`accepted_count` counts all the caller's accepted relationships, not the page
or pending requests. Rows, count, and pagination metadata in one response come
from a consistent database snapshot. Separate pages are live reads: concurrent
changes may change counts, and new rows above the cursor appear after refresh.

### Error envelope

Preserve the auth/profile error shape, adding optional `details` for domain
information. Omit `details` when inapplicable. Clients branch on `code`.

```json
{
  "error": {
    "code": "CONNECTION_LIMIT_REACHED",
    "message": "This connection cannot be accepted because the connection limit has been reached.",
    "details": { "limit": 150 }
  }
}
```

Do not return the other participant's count or identify which participant is
full. The caller can consult their own People metadata. Pending conflicts
expose only `details.request_id` for a relationship involving the caller.

In addition to the domain codes above, `400 INVALID_REQUEST` covers malformed
JSON, unknown/missing fields, malformed route UUIDs, invalid query/cursor
values, and unexpected bodies. `422 VALIDATION_FAILED` covers supplied
recipient fields with an invalid type or UUID (optional
`fields.recipient_id`). Unexpected failures return `500 INTERNAL_ERROR`
without database details.

## Atomicity and the 150-person invariant

All pair mutations (send, accept, reject, remove, and future safety mutations)
must lock both participant `public.users` rows in ascending UUID order in a
transaction. Re-read the resource and verify authorization after acquiring
locks. Every mutation touching a user shares this serialization boundary.

For a new acceptance, under PostgreSQL READ COMMITTED:

1. Acquire both participant row locks in canonical order.
2. Re-read pending state and current permission; handle resolved retries first.
3. Count each participant's accepted connections using fresh statements after
   locks are held. If either count is >=150, return conflict without mutation.
4. Insert one canonical connection and resolve its request in the transaction.
5. Commit before returning success.

Do not count before waiting for locks or rely on a stale transaction snapshot.
An alternative isolation strategy must provide equivalent guarantees and
transaction retries where required. All accepted-connection creation paths,
including later invite paths, must use the same domain boundary. Database
uniqueness is the final duplicate defense; a row CHECK alone cannot enforce
the cross-row capacity limit.

At 149 with concurrent acceptances sharing a user, at most one adds a
connection unless a serialized removal first frees capacity. Failure must not
leave a newly accepted request without its connection or any partial
reciprocal relationship. Removal and acceptance use the same lock protocol.

## Mobile behavior and acceptance tests

People renders accepted connections and a quiet caller-only `x / 150` count.
Incoming requests expose accept/reject. Crossing-send conflicts refresh
incoming requests so the caller explicitly decides. Disable duplicate
in-flight actions. Refresh both lists/count after successful mutations or
stale-state conflicts; do not permanently adjust a local count by assumption.

Mocks/UI tests cover loading, empty, populated, pagination, network failure,
auth/onboarding failure, resolved requests, already-absent removal, capacity
conflicts, and reserved forbidden responses. Full callers can still
reject/remove and view incoming requests. A capacity conflict may occur even
when the caller's displayed count is below 150.

Backend and integration acceptance criteria:

- Send → incoming → accept makes each person appear in the other's People list.
- Rejection creates no relationship; removal updates both lists and counts.
- Self, duplicate, crossing, retries, and fresh requests match this contract.
- Stranger and sender decision attempts fail without revealing request details.
- Strangers cannot list another person's graph or remove their connections.
- Forged caller identity, missing/invalid tokens, and missing profiles fail.
- Acceptance succeeds at 149 and fails at 150 for either participant.
- Real PostgreSQL concurrency tests never produce 151, including shared
  senders, shared recipients, and both participants near capacity.
- Concurrent same-pair sends create at most one pending request; concurrent
  accepts create one connection. Accept/reject and remove/accept races follow
  serialized lifecycle rules. Failed transactions preserve pending state/counts.
- Equal timestamps paginate deterministically; empty/final pages and total
  counts match. Cross-caller/list cursors are rejected.
- Direct client database access cannot bypass the API. Safety integration
  later adds both block directions and block/accept races before beta.

Backend implementations run `go test ./...`, `go vet ./...`, formatting checks,
and PostgreSQL integration tests. Mobile runs configured typecheck/UI tests.
Prefer root `make test` for combined verification. This documentation does not
establish that runtime requirements already pass.

## Out of scope

Implementation, applied migrations, invitation/deep-link lifecycle, public
discovery, outgoing-request management, cancellation, notification delivery,
avatar download URLs, block/report/account-deletion policy, and feed
implementation remain separate tasks. No follower mechanics, engagement
ranking, or network expansion beyond 150 accepted connections is introduced.
