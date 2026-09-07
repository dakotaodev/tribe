import "react-native-url-polyfill/auto";

import { createClient, type SupabaseClient } from "@supabase/supabase-js";
import * as SecureStore from "expo-secure-store";

const supabaseUrl = process.env.EXPO_PUBLIC_SUPABASE_URL;
const supabasePublishableKey = process.env.EXPO_PUBLIC_SUPABASE_PUBLISHABLE_KEY;

function requiredPublicConfig(name: string, value: string | undefined): string {
  if (!value || value.startsWith("https://your-project") || value === "your-publishable-key") {
    throw new Error(`${name} must be configured with a public Supabase project value.`);
  }

  return value;
}

export function createSupabaseClient(): SupabaseClient {
  return createClient(
    requiredPublicConfig("EXPO_PUBLIC_SUPABASE_URL", supabaseUrl),
    requiredPublicConfig("EXPO_PUBLIC_SUPABASE_PUBLISHABLE_KEY", supabasePublishableKey),
    {
      auth: {
        autoRefreshToken: true,
        detectSessionInUrl: false,
        persistSession: true,
        storage: {
          getItem: (key) => SecureStore.getItemAsync(key),
          removeItem: (key) => SecureStore.deleteItemAsync(key),
          setItem: (key, value) => SecureStore.setItemAsync(key, value)
        }
      }
    }
  );
}

let supabase: SupabaseClient | null = null;
let supabaseConfigurationError: Error | null = null;

try {
  supabase = createSupabaseClient();
} catch (error) {
  supabaseConfigurationError = error instanceof Error ? error : new Error("Supabase configuration is invalid.");
}

export { supabase, supabaseConfigurationError };
