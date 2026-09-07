import * as Linking from "expo-linking";
import { AppState } from "react-native";
import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type PropsWithChildren } from "react";
import type { Session } from "@supabase/supabase-js";

import { createSessionController, type SessionState } from "../lib/session-controller";
import { supabase, supabaseConfigurationError } from "../lib/supabase";

type SessionContextValue = SessionState & {
  sendMagicLink: (email: string) => Promise<void>;
  signInWithApple: () => Promise<void>;
  signOut: () => Promise<void>;
};

const unavailable = async () => {
  throw new Error("Supabase is not configured.");
};

const SessionContext = createContext<SessionContextValue>({
  error: null,
  isLoading: true,
  sendMagicLink: unavailable,
  session: null,
  signInWithApple: unavailable,
  signOut: unavailable
});

const callbackUrl = Linking.createURL("auth/callback", { scheme: "tribe" });

function isAuthCallback(url: string): boolean {
  return url.startsWith(callbackUrl);
}

export function SessionProvider({ children }: PropsWithChildren) {
  const [state, setState] = useState<SessionState>({
    error: supabaseConfigurationError,
    isLoading: supabaseConfigurationError === null,
    session: null
  });
  const controllerRef = useRef<ReturnType<typeof createSessionController> | null>(null);

  useEffect(() => {
    const client = supabase;
    if (!client) {
      return;
    }

    const controller = createSessionController({
      callbackUrl,
      client,
      onStateChange: setState
    });
    controllerRef.current = controller;
    const subscription = controller.subscribe();
    void controller.initialize();
    void Linking.getInitialURL().then((url) => {
      if (url && isAuthCallback(url)) {
        void controller.handleCallback(url);
      }
    });
    const linkingSubscription = Linking.addEventListener("url", ({ url }) => {
      if (isAuthCallback(url)) {
        void controller.handleCallback(url);
      }
    });
    const appStateSubscription = AppState.addEventListener("change", (nextState) => {
      if (nextState === "active") {
        client.auth.startAutoRefresh();
      } else {
        client.auth.stopAutoRefresh();
      }
    });
    client.auth.startAutoRefresh();

    return () => {
      controllerRef.current = null;
      subscription.unsubscribe();
      linkingSubscription.remove();
      appStateSubscription.remove();
      client.auth.stopAutoRefresh();
    };
  }, []);

  const requireController = useCallback(() => {
    if (!controllerRef.current) {
      throw supabaseConfigurationError ?? new Error("Session setup is still loading.");
    }
    return controllerRef.current;
  }, []);

  const value = useMemo<SessionContextValue>(() => ({
    ...state,
    sendMagicLink: (email) => requireController().sendMagicLink(email),
    signInWithApple: () => requireController().signInWithApple(),
    signOut: () => requireController().signOut()
  }), [requireController, state]);

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): SessionContextValue {
  return useContext(SessionContext);
}

export type { Session };
