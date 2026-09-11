const assert = require("node:assert/strict");
const test = require("node:test");
const { ApiError, ProfileApi } = require("../.test-dist/api.js");
const token = "authenticated-token";

test("GET /me missing-profile contract enters onboarding", async () => {
  const requests = [];
  const api = new ProfileApi("https://api.test", async (url, init) => {
    requests.push({ url, init });
    return new Response(JSON.stringify({ error: { code: "PROFILE_ONBOARDING_REQUIRED", message: "Create your profile to continue." } }), { status: 404 });
  });
  assert.equal(await api.getMe(token), null);
  assert.equal(requests[0].url, "https://api.test/me");
  assert.equal(requests[0].init.headers.Authorization, `Bearer ${token}`);
});

test("GET /me parses the standard profile envelope", async () => {
  const api = new ProfileApi("https://api.test", async () => new Response(JSON.stringify({ data: {
    id: "user-id",
    username: "dakota",
    display_name: "Dakota",
    bio: "Hello",
    avatar_url: null,
    created_at: "2026-09-11T00:00:00Z",
    updated_at: "2026-09-11T00:00:00Z"
  } }), { status: 200 }));
  assert.deepEqual(await api.getMe(token), {
    id: "user-id",
    username: "dakota",
    displayName: "Dakota",
    bio: "Hello",
    avatarUrl: null,
    createdAt: "2026-09-11T00:00:00Z",
    updatedAt: "2026-09-11T00:00:00Z"
  });
});

test("GET /me does not treat an unrelated 404 as onboarding", async () => {
  const api = new ProfileApi("https://api.test", async () => new Response(JSON.stringify({ error: {
    code: "RESOURCE_NOT_FOUND",
    message: "Route not found."
  } }), { status: 404 }));
  await assert.rejects(api.getMe(token), (error) => error instanceof ApiError && error.kind === "notFound" && error.code === "RESOURCE_NOT_FOUND");
});

test("PUT /me maps duplicate username and never sends a user ID", async () => {
  let request;
  const api = new ProfileApi("https://api.test", async (_url, init) => {
    request = init;
    return new Response(JSON.stringify({ error: { code: "USERNAME_TAKEN", message: "That username is unavailable.", fields: { username: "Choose a different username." } } }), { status: 409 });
  });
  await assert.rejects(api.updateMe(token, { username: "  Dakota_1 ", displayName: " Dakota ", bio: " Hello " }), (error) => Boolean(error instanceof ApiError && error.kind === "conflict" && error.fields.username));
  assert.deepEqual(JSON.parse(request.body), { username: "dakota_1", display_name: "Dakota", bio: "Hello" });
  assert.equal(request.body.includes("user_id"), false);
});

test("network failures produce a retryable user-facing error", async () => {
  const api = new ProfileApi("https://api.test", async () => { throw new TypeError("offline"); });
  await assert.rejects(api.getMe(token), (error) => error instanceof ApiError && error.kind === "network" && error.message.includes("connection"));
});
