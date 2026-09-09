import "react-native-url-polyfill/auto";

import * as AppleAuthentication from "expo-apple-authentication";
import * as Crypto from "expo-crypto";
import type { Session } from "@supabase/supabase-js";

export type SessionState = {
  error: Error | null;
  isLoading: boolean;
  session: Session | null;
};

type AuthChangeCallback = (_event: string, session: Session | null) => void;

export type AuthClient = {
  auth: {
    exchangeCodeForSession: (code: string) => Promise<{ error: Error | null }>;
    getSession: () => Promise<{ data: { session: Session | null }; error: Error | null }>;
    onAuthStateChange: (callback: AuthChangeCallback) => {
      data: { subscription: { unsubscribe: () => void } };
    };
    setSession: (tokens: { access_token: string; refresh_token: string }) => Promise<{ error: Error | null }>;
    signInWithIdToken: (credentials: {
      nonce: string;
      provider: "apple";
      token: string;
    }) => Promise<{ error: Error | null }>;
    signInWithOtp: (credentials: {
      email: string;
      options: { emailRedirectTo: string };
    }) => Promise<{ error: Error | null }>;
    signOut: () => Promise<{ error: Error | null }>;
  };
};

export type AppleAuthenticator = {
  isAvailableAsync: () => Promise<boolean>;
  signInAsync: (options: {
    nonce: string;
    requestedScopes: AppleAuthentication.AppleAuthenticationScope[];
  }) => Promise<{ identityToken: string | null }>;
};

type ControllerDependencies = {
  apple?: AppleAuthenticator;
  callbackUrl: string;
  client: AuthClient;
  digestNonce?: (nonce: string) => Promise<string>;
  newNonce?: () => string;
  onStateChange: (state: SessionState) => void;
};

export function createSessionController({
  apple = AppleAuthentication,
  callbackUrl,
  client,
  digestNonce = (nonce) => Crypto.digestStringAsync(Crypto.CryptoDigestAlgorithm.SHA256, nonce),
  newNonce = Crypto.randomUUID,
  onStateChange
}: ControllerDependencies) {
  let session: Session | null = null;
  const publish = (next: SessionState) => {
    session = next.session;
    onStateChange(next);
  };
  const setError = (error: Error | null) => publish({ error, isLoading: false, session });
  const clearError = () => publish({ error: null, isLoading: false, session });

  const requireNoError = (error: Error | null) => {
    if (error) {
      throw error;
    }
  };

  return {
    async handleCallback(url: string) {
      const query = url.includes("#") ? url.slice(url.indexOf("#") + 1) : url.slice(url.indexOf("?") + 1);
      const values = new URLSearchParams(query);
      const code = values.get("code");
      const accessToken = values.get("access_token");
      const refreshToken = values.get("refresh_token");

      try {
        if (code) {
          const { error } = await client.auth.exchangeCodeForSession(code);
          requireNoError(error);
          return;
        }

        if (accessToken && refreshToken) {
          const { error } = await client.auth.setSession({
            access_token: accessToken,
            refresh_token: refreshToken
          });
          requireNoError(error);
          return;
        }

        throw new Error("The sign-in link did not contain a session.");
      } catch (error) {
        setError(error instanceof Error ? error : new Error("Unable to complete sign-in."));
      }
    },

    async initialize() {
      const { data, error } = await client.auth.getSession();
      publish({ error, isLoading: false, session: data.session });
    },

    subscribe() {
      return client.auth.onAuthStateChange((_event, session) => {
        publish({ error: null, isLoading: false, session });
      }).data.subscription;
    },

    async sendMagicLink(email: string) {
      try {
        const { error } = await client.auth.signInWithOtp({
          email,
          options: { emailRedirectTo: callbackUrl }
        });
        requireNoError(error);
        clearError();
      } catch (error) {
        setError(error instanceof Error ? error : new Error("Unable to send a sign-in link."));
        throw error;
      }
    },

    async signInWithApple() {
      try {
        if (!(await apple.isAvailableAsync())) {
          throw new Error("Sign in with Apple is not available on this device.");
        }

        const nonce = newNonce();
        const credential = await apple.signInAsync({
          nonce: await digestNonce(nonce),
          requestedScopes: [AppleAuthentication.AppleAuthenticationScope.FULL_NAME, AppleAuthentication.AppleAuthenticationScope.EMAIL]
        });

        if (!credential.identityToken) {
          throw new Error("Apple did not return an identity token.");
        }

        const { error } = await client.auth.signInWithIdToken({
          nonce,
          provider: "apple",
          token: credential.identityToken
        });
        requireNoError(error);
        clearError();
      } catch (error) {
        setError(error instanceof Error ? error : new Error("Unable to sign in with Apple."));
        throw error;
      }
    },

    async signOut() {
      try {
        const { error } = await client.auth.signOut();
        requireNoError(error);
        publish({ error: null, isLoading: false, session: null });
      } catch (error) {
        setError(error instanceof Error ? error : new Error("Unable to sign out."));
        throw error;
      }
    }
  };
}
