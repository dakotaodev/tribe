function requiredEnvironment(name: string, value: string | undefined): string {
  const configured = value?.trim();
  if (!configured) {
    throw new Error(`${name} is required. Copy mobile/.env.example to mobile/.env.`);
  }
  return configured.replace(/\/$/, "");
}

export const environment = {
  apiUrl: requiredEnvironment("EXPO_PUBLIC_API_URL", process.env.EXPO_PUBLIC_API_URL),
  supabaseUrl: requiredEnvironment("EXPO_PUBLIC_SUPABASE_URL", process.env.EXPO_PUBLIC_SUPABASE_URL),
  supabaseKey: requiredEnvironment(
    "EXPO_PUBLIC_SUPABASE_PUBLISHABLE_KEY",
    process.env.EXPO_PUBLIC_SUPABASE_PUBLISHABLE_KEY
  )
};
