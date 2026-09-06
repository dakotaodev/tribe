# Tribe — MVP Scope

## Objective

Build the smallest usable version of Tribe capable of testing whether small real-world social groups will repeatedly use the product.

The MVP is an experiment, not a complete social network.

## Primary Hypothesis

Groups of approximately 5–15 people who know one another in real life will:

1. join Tribe
2. establish reciprocal connections
3. post personal updates
4. interact with other people's updates
5. return without external prompting
6. invite additional people themselves

## Core MVP Loop

```text
sign up
→ create profile
→ invite/connect
→ post
→ friend sees post
→ friend interacts
→ return
```

Every MVP feature must support this loop or a basic safety/privacy requirement.

## Required Features

### Authentication

Users must be able to:

- create an account
- sign in
- sign out
- remain authenticated between sessions
- delete their account

Initial authentication may use:

- Sign in with Apple
- email-based authentication

Authentication should use Supabase Auth.

### Profiles

Each user has:

- unique username
- display name
- avatar
- optional short bio
- created timestamp

Public user discovery is not required.

### Reciprocal connections

Users can:

- invite another person
- send a connection request
- receive a connection request
- accept a request
- reject a request
- remove an existing connection

Connections are reciprocal.

There is no follower model.

### 150-person limit

A user may have at most:

**150 accepted connections**

The backend must enforce this invariant.

Acceptance must fail if either participant already has 150 accepted connections.

Concurrency must not allow a user to exceed the limit.

### Posts

Users can create:

- text posts
- photo posts
- text + photo posts

MVP constraints:

- maximum 4 photos per post
- no video
- reasonable text length limit
- users may delete their own posts

Editing may be added if inexpensive but is not required for initial validation.

### Feed

Users see posts from:

- themselves
- accepted connections

Users must not see posts from:

- strangers
- pending connection requests
- blocked users

Feed behavior:

- reverse chronological
- cursor paginated
- no ranking
- no recommendations

### End-of-feed state

When the user reaches the end of available content, show an explicit state such as:

> You're caught up.

No recommended posts should appear afterward.

### Comments

Connections may comment on visible posts.

Users can delete their own comments.

### Lightweight reactions

Support one simple reaction initially.

Example:

❤️

Do not build elaborate reaction systems.

Avoid prominently displaying aggregate popularity metrics.

### Invitations

Users need an easy way to invite someone outside the application.

Ideal flow:

```text
user creates invite
→ shares through iMessage/share sheet
→ recipient opens link
→ installs/opens Tribe
→ creates account
→ invitation is preserved
→ connection request is immediately available
```

The exact deep-link flow can be simplified during early development if necessary.

### Blocking

Users can block another user.

A block should prevent inappropriate future interaction and content visibility according to the final backend rules.

### Reporting

Users must be able to report:

- another user
- a post

For the private beta, the moderation workflow may be manual.

### Basic analytics

Record product events necessary to evaluate the experiment.

At minimum:

```text
account_created
session_started

invite_created
invite_opened

connection_request_sent
connection_request_accepted

post_created
comment_created
reaction_created

feed_caught_up
```

## MVP Screens

Initial mobile screens:

1. Welcome
2. Authentication
3. Profile setup
4. Home/feed
5. Create post
6. Connections
7. Connection requests
8. Profile
9. Settings
10. Report/block flows

Do not create additional navigation surfaces without a product requirement.

## Initial Navigation

Prefer a minimal structure.

Likely:

```text
Home      Create      People
```

Profile/settings may initially be accessed through the user's avatar.

This is not a rigid requirement if implementation testing reveals a simpler navigation model.

## Explicit Non-Goals

Do not build the following before MVP validation:

- direct messages
- video
- stories
- reels
- hashtags
- public profiles
- public follower mechanics
- contact recommendations
- algorithmic recommendations
- trending content
- reposts
- quote posts
- creator accounts
- business accounts
- advertising
- subscriptions
- Android-specific optimization
- web application
- sophisticated relationship scoring
- AI features
- Dunbar relationship tiers
- family/group circles
- events
- shared albums
- Redis
- Kafka
- background job infrastructure unless strictly required
- microservices
- Kubernetes

## Validation Cohorts

The private beta should eventually include at least three small networks.

### Cohort A

Family or household network.

Approximately 5–10 people.

### Cohort B

Friend group.

Approximately 5–10 people.

### Cohort C

Independent social graph where the founder is not central.

Approximately 5–10 people.

Cohort C is particularly important because founder-driven activity can create false engagement signals.

## Early Metrics

### Activation

Within approximately 72 hours, determine whether a new user:

- creates or accepts several connections
- views social content
- creates or interacts with at least one post

Initial working activation definition:

```text
3+ accepted connections
AND
1+ post/comment/reaction
```

This definition may be adjusted after observing actual behavior.

### D1 retention

Did the user return the following day?

### D7 retention

Did the user return approximately one week later?

### D30 retention

Did the user continue using the product approximately one month later?

### Weekly posting

What percentage of activated users create at least one post during a week?

### Organic invitation rate

Do existing users invite additional people without being directly prompted by the founder?

This is a critical early signal.

## Qualitative Signals

Strong signals include users saying:

- "I wanted to post this here instead of Instagram."
- "Can you get Sarah on here?"
- "Why wasn't the app working yesterday?"
- "I wish my family used this."
- "I actually saw everything people posted."

Weak signals include:

- "Cool idea."
- "I hate Instagram too."
- "I'd definitely use something like this."

Behavior matters more than stated enthusiasm.

## Initial Continue/Pivot/Kill Decision

After approximately 30 days of meaningful beta usage:

### Continue

Evidence includes:

- users post repeatedly
- multiple independent groups remain active
- invitations occur organically
- users return without reminders
- content appears that users deliberately chose not to post publicly

### Pivot

Users value a specific behavior but not the overall feed/network concept.

Examples:

- family sharing
- private photo sharing
- relationship reminders
- small-group communication

### Stop

Users like the philosophy but consistently fail to use the product after novelty fades.

## MVP Rule

Until at least 10 real people are actively using the current version:

**Do not add speculative product features.**