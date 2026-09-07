import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Session } from "@supabase/supabase-js";

vi.mock("expo-apple-authentication", () => ({
  AppleAuthenticationScope: { EMAIL: "email", FULL_NAME: "full_name" }
}));
vi.mock("react-native-url-polyfill/auto", () => ({}));
vi.mock("expo-crypto", () => ({
  CryptoDigestAlgorithm: { SHA256: "SHA-256" },
  digestStringAsync: vi.fn(),
  randomUUID: vi.fn()
}));

import { createSessionController, type AuthClient, type SessionState } from "./session-controller";

const restoredSession = {
  access_token: "access-token",
  refresh_token: "refresh-token",
  user: { id: "user-id", aud: "authenticated", created_at: "2026-01-01T00:00:00Z" }
} as Session;

function createClient(overrides: Partial<AuthClient["auth"]> = {}) {
  let listener: ((event: string, session: Session | null) => void) | undefined;
  const auth = {
    exchangeCodeForSession: vi.fn().mockResolvedValue({ error: null }),
    getSession: vi.fn().mockResolvedValue({ data: { session: restoredSession }, error: null }),
    onAuthStateChange: vi.fn((callback) => {
      listener = callback;
      return { data: { subscription: { unsubscribe: vi.fn() } } };
    }),
    setSession: vi.fn().mockResolvedValue({ error: null }),
    signInWithIdToken: vi.fn().mockResolvedValue({ error: null }),
    signInWithOtp: vi.fn().mockResolvedValue({ error: null }),
    signOut: vi.fn().mockResolvedValue({ error: null }),
    ...overrides
  };

  return { auth, emit: (event: string, session: Session | null) => listener?.(event, session) };
}

describe("session controller", () => {
  let states: SessionState[];

  beforeEach(() => {
    states = [];
  });

  function controllerFor(
    client: AuthClient,
    overrides: Partial<Omit<Parameters<typeof createSessionController>[0], "callbackUrl" | "client" | "onStateChange">> = {}
  ) {
    return createSessionController({
      apple: {
        isAvailableAsync: vi.fn().mockResolvedValue(true),
        signInAsync: vi.fn().mockResolvedValue({ identityToken: "apple-token" })
      },
      callbackUrl: "tribe://auth/callback",
      client,
      digestNonce: vi.fn().mockResolvedValue("hashed-nonce"),
      newNonce: vi.fn().mockReturnValue("raw-nonce"),
      onStateChange: (state) => states.push(state),
      ...overrides
    });
  }

  it("restores a persisted session and observes later auth changes", async () => {
    const client = createClient();
    const controller = controllerFor(client);

    controller.subscribe();
    await controller.initialize();
    client.emit("SIGNED_OUT", null);

    expect(states).toEqual([
      { error: null, isLoading: false, session: restoredSession },
      { error: null, isLoading: false, session: null }
    ]);
  });

  it("sends an email magic link to the registered callback", async () => {
    const client = createClient();

    await controllerFor(client).sendMagicLink("person@example.com");

    expect(client.auth.signInWithOtp).toHaveBeenCalledWith({
      email: "person@example.com",
      options: { emailRedirectTo: "tribe://auth/callback" }
    });
  });

  it("surfaces magic-link errors without discarding the current session", async () => {
    const client = createClient({ signInWithOtp: vi.fn().mockResolvedValue({ error: new Error("Rate limited") }) });
    const controller = controllerFor(client);
    await controller.initialize();

    await expect(controller.sendMagicLink("person@example.com")).rejects.toThrow("Rate limited");

    expect(states.at(-1)).toEqual({ error: new Error("Rate limited"), isLoading: false, session: restoredSession });
  });

  it("exchanges a PKCE callback code and reports malformed callbacks", async () => {
    const client = createClient();
    const controller = controllerFor(client);

    await controller.handleCallback("tribe://auth/callback?code=one-time-code");
    await controller.handleCallback("tribe://auth/callback");

    expect(client.auth.exchangeCodeForSession).toHaveBeenCalledWith("one-time-code");
    expect(states.at(-1)?.error?.message).toBe("The sign-in link did not contain a session.");
  });

  it("uses the native Apple token flow with a hashed nonce", async () => {
    const client = createClient();
    const apple = {
      isAvailableAsync: vi.fn().mockResolvedValue(true),
      signInAsync: vi.fn().mockResolvedValue({ identityToken: "apple-token" })
    };

    await controllerFor(client, { apple }).signInWithApple();

    expect(apple.signInAsync).toHaveBeenCalledWith({
      nonce: "hashed-nonce",
      requestedScopes: ["full_name", "email"]
    });
    expect(client.auth.signInWithIdToken).toHaveBeenCalledWith({
      nonce: "raw-nonce",
      provider: "apple",
      token: "apple-token"
    });
  });

  it("surfaces Apple errors", async () => {
    const client = createClient();
    const controller = controllerFor(client, {
      apple: { isAvailableAsync: vi.fn().mockResolvedValue(false), signInAsync: vi.fn() }
    });

    await expect(controller.signInWithApple()).rejects.toThrow("not available");

    expect(states.at(-1)?.error?.message).toContain("not available");
  });

  it("clears the local session after sign-out and surfaces failures", async () => {
    const client = createClient();
    const controller = controllerFor(client);
    await controller.initialize();
    await controller.signOut();

    expect(states.at(-1)).toEqual({ error: null, isLoading: false, session: null });

    const failingClient = createClient({ signOut: vi.fn().mockResolvedValue({ error: new Error("Offline") }) });
    await expect(controllerFor(failingClient).signOut()).rejects.toThrow("Offline");
    expect(states.at(-1)?.error?.message).toBe("Offline");
  });
});
