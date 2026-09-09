const assert = require("node:assert/strict");
const test = require("node:test");
const { ApiError, ProfileApi } = require("../.test-dist/api.js");
const token = "authenticated-token";

test("GET /me missing-profile contract enters onboarding", async () => {
  const requests = [];
  const api = new ProfileApi("https://api.test", async (url, init) => {
    requests.push({ url, init });
    return new Response(JSON.stringify({ error: { code: "profile_not_found", message: "Complete your profile." } }), { status: 404 });
  });
  assert.equal(await api.getMe(token), null);
  assert.equal(requests[0].url, "https://api.test/me");
  assert.equal(requests[0].init.headers.Authorization, `Bearer ${token}`);
});

test("PUT /me maps duplicate username and never sends a user ID", async () => {
  let request;
  const api = new ProfileApi("https://api.test", async (_url, init) => {
    request = init;
    return new Response(JSON.stringify({ error: { code: "username_conflict", message: "That username is already taken." } }), { status: 409 });
  });
  await assert.rejects(api.updateMe(token, { username: "  Dakota_1 ", displayName: " Dakota ", bio: " Hello " }), (error) => Boolean(error instanceof ApiError && error.kind === "conflict" && error.fields.username));
  assert.deepEqual(JSON.parse(request.body), { username: "dakota_1", display_name: "Dakota", bio: "Hello" });
  assert.equal(request.body.includes("user_id"), false);
});

test("network failures produce a retryable user-facing error", async () => {
  const api = new ProfileApi("https://api.test", async () => { throw new TypeError("offline"); });
  await assert.rejects(api.getMe(token), (error) => error instanceof ApiError && error.kind === "network" && error.message.includes("connection"));
});
