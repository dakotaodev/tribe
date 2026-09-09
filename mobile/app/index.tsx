import { StatusBar } from "expo-status-bar";
import { useEffect, useMemo, useState } from "react";
import { ActivityIndicator, KeyboardAvoidingView, Platform, Pressable, SafeAreaView, ScrollView, StyleSheet, Text, TextInput, View } from "react-native";
import { ApiError, ProfileApi } from "../src/api";
import { environment } from "../src/config";
import { type Profile, type ProfileErrors, type ProfileInput, validateProfile } from "../src/profile";
import { type Session, SupabaseSessionClient } from "../src/session";

type Route = "loading" | "loadError" | "auth" | "onboarding" | "profile" | "edit";
const blankProfile: ProfileInput = { username: "", displayName: "", bio: "" };

export default function App() {
  const auth = useMemo(() => new SupabaseSessionClient(environment.supabaseUrl, environment.supabaseKey), []);
  const api = useMemo(() => new ProfileApi(environment.apiUrl), []);
  const [route, setRoute] = useState<Route>("loading");
  const [session, setSession] = useState<Session | null>(null);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [error, setError] = useState("");

  async function loadProfile(nextSession: Session) {
    try {
      const nextProfile = await api.getMe(nextSession.accessToken);
      setProfile(nextProfile);
      setRoute(nextProfile ? "profile" : "onboarding");
    } catch (caught) {
      if (caught instanceof ApiError && caught.kind === "unauthorized") return signOut();
      setError(caught instanceof Error ? caught.message : "Something went wrong.");
      setRoute("loadError");
    }
  }

  useEffect(() => { void auth.restore().then((restored) => { setSession(restored); restored ? void loadProfile(restored) : setRoute("auth"); }).catch(() => setRoute("auth")); }, []);

  async function signedIn(nextSession: Session) { setError(""); setSession(nextSession); setRoute("loading"); await loadProfile(nextSession); }
  async function signOut() { await auth.signOut(session); setSession(null); setProfile(null); setError(""); setRoute("auth"); }

  if (route === "loading") return <Loading />;
  if (route === "loadError" && session) return <MessageScreen message={error} onRetry={() => { setRoute("loading"); void loadProfile(session); }} onSignOut={() => void signOut()} />;
  if (route === "auth") return <AuthScreen auth={auth} error={error} onAuthenticated={signedIn} />;
  if (route === "onboarding" && session) return <ProfileForm title="Make Tribe yours" subtitle="Choose how your people will know you." initial={blankProfile} token={session.accessToken} api={api} submitLabel="Create profile" onSaved={(value) => { setProfile(value); setRoute("profile"); }} />;
  if (route === "edit" && session && profile) return <ProfileForm title="Edit profile" subtitle="Keep it simple and recognizable." initial={profile} token={session.accessToken} api={api} submitLabel="Save changes" onCancel={() => setRoute("profile")} onSaved={(value) => { setProfile(value); setRoute("profile"); }} />;
  if (profile) return <ProfileScreen profile={profile} onEdit={() => setRoute("edit")} onSignOut={() => void signOut()} />;
  return <Loading />;
}

function Screen({ children }: { children: React.ReactNode }) { return <SafeAreaView style={styles.safe}><StatusBar style="dark" /><KeyboardAvoidingView style={styles.flex} behavior={Platform.OS === "ios" ? "padding" : undefined}><ScrollView keyboardShouldPersistTaps="handled" contentContainerStyle={styles.screen}>{children}</ScrollView></KeyboardAvoidingView></SafeAreaView>; }
function Loading() { return <SafeAreaView style={[styles.safe, styles.center]}><StatusBar style="dark" /><ActivityIndicator color="#5E6B4A" /><Text style={styles.muted}>Opening your Tribe…</Text></SafeAreaView>; }
function MessageScreen({ message, onRetry, onSignOut }: { message: string; onRetry(): void; onSignOut(): void }) { return <Screen><View><Text style={styles.eyebrow}>YOUR TRIBE</Text><Text style={styles.hero}>We couldn’t load your profile.</Text><Text style={styles.copy}>{message}</Text></View><View style={styles.card}><Button label="Try again" onPress={onRetry} /><Pressable onPress={onSignOut}><Text style={styles.signOut}>Sign out</Text></Pressable></View></Screen>; }

function AuthScreen({ auth, error: initialError, onAuthenticated }: { auth: SupabaseSessionClient; error: string; onAuthenticated(session: Session): Promise<void> }) {
  const [mode, setMode] = useState<"signin" | "signup">("signin"); const [email, setEmail] = useState(""); const [password, setPassword] = useState(""); const [error, setError] = useState(initialError); const [busy, setBusy] = useState(false);
  async function submit() {
    if (!email.trim() || !email.includes("@")) return setError("Enter a valid email address.");
    if (password.length < 6) return setError("Password must be at least 6 characters.");
    setBusy(true); setError("");
    try { await onAuthenticated(await (mode === "signin" ? auth.signIn(email, password) : auth.signUp(email, password))); } catch (caught) { setError(caught instanceof Error ? caught.message : "Authentication failed."); } finally { setBusy(false); }
  }
  return <Screen><View style={styles.brand}><Text style={styles.eyebrow}>TRIBE</Text><Text style={styles.hero}>A private place for your people.</Text><Text style={styles.copy}>No followers. No noise. Just the people you care about.</Text></View><View style={styles.card}><Text style={styles.heading}>{mode === "signin" ? "Welcome back" : "Create your account"}</Text><Field label="Email" value={email} onChangeText={setEmail} autoCapitalize="none" keyboardType="email-address" /><Field label="Password" value={password} onChangeText={setPassword} secureTextEntry /><ErrorText value={error} /><Button label={busy ? "Please wait…" : mode === "signin" ? "Sign in" : "Sign up"} disabled={busy} onPress={() => void submit()} /><Pressable accessibilityRole="button" onPress={() => { setMode(mode === "signin" ? "signup" : "signin"); setError(""); }}><Text style={styles.link}>{mode === "signin" ? "New to Tribe? Create an account" : "Already have an account? Sign in"}</Text></Pressable></View></Screen>;
}

function ProfileForm({ title, subtitle, initial, token, api, submitLabel, onSaved, onCancel }: { title: string; subtitle: string; initial: ProfileInput; token: string; api: ProfileApi; submitLabel: string; onSaved(profile: Profile): void; onCancel?: () => void }) {
  const [value, setValue] = useState<ProfileInput>(initial); const [errors, setErrors] = useState<ProfileErrors>({}); const [message, setMessage] = useState(""); const [busy, setBusy] = useState(false);
  async function submit() {
    const nextErrors = validateProfile(value); setErrors(nextErrors); setMessage(""); if (Object.keys(nextErrors).length) return;
    setBusy(true); try { onSaved(await api.updateMe(token, value)); } catch (caught) { if (caught instanceof ApiError) { setErrors(caught.fields); setMessage(caught.message); } else setMessage("Something went wrong."); } finally { setBusy(false); }
  }
  return <Screen><View style={styles.top}><Text style={styles.eyebrow}>{onCancel ? "YOUR PROFILE" : "WELCOME TO TRIBE"}</Text><Text style={styles.hero}>{title}</Text><Text style={styles.copy}>{subtitle}</Text></View><View style={styles.card}><Field label="Username" hint="Lowercase letters, numbers, and underscores" value={value.username} onChangeText={(username) => setValue({ ...value, username })} autoCapitalize="none" error={errors.username} /><Field label="Display name" value={value.displayName} onChangeText={(displayName) => setValue({ ...value, displayName })} error={errors.displayName} /><Field label="Bio (optional)" value={value.bio} onChangeText={(bio) => setValue({ ...value, bio })} multiline maxLength={500} error={errors.bio} /><Text style={styles.counter}>{value.bio.length}/500</Text><ErrorText value={message} /><Button label={busy ? "Saving…" : submitLabel} disabled={busy} onPress={() => void submit()} />{onCancel && <Pressable onPress={onCancel}><Text style={styles.link}>Cancel</Text></Pressable>}</View></Screen>;
}

function ProfileScreen({ profile, onEdit, onSignOut }: { profile: Profile; onEdit(): void; onSignOut(): void }) { const initials = profile.displayName.split(/\s+/).map((part) => part[0]).join("").slice(0, 2).toUpperCase(); return <Screen><View style={styles.profileHeader}><View style={styles.avatar}><Text style={styles.initials}>{initials}</Text></View><Text style={styles.hero}>{profile.displayName}</Text><Text style={styles.handle}>@{profile.username}</Text>{profile.bio ? <Text style={styles.bio}>{profile.bio}</Text> : <Text style={styles.muted}>A quiet corner for the people who matter.</Text>}</View><View style={styles.card}><Button label="Edit profile" onPress={onEdit} /><Pressable onPress={onSignOut}><Text style={styles.signOut}>Sign out</Text></Pressable></View></Screen>; }

type FieldProps = React.ComponentProps<typeof TextInput> & { label: string; hint?: string; error?: string };
function Field({ label, hint, error, multiline, ...props }: FieldProps) { return <View style={styles.field}><Text style={styles.label}>{label}</Text>{hint && <Text style={styles.hint}>{hint}</Text>}<TextInput accessibilityLabel={label} style={[styles.input, multiline && styles.textarea, error && styles.inputError]} placeholderTextColor="#9A9287" multiline={multiline} {...props} />{error && <Text style={styles.fieldError}>{error}</Text>}</View>; }
function Button({ label, onPress, disabled }: { label: string; onPress(): void; disabled?: boolean }) { return <Pressable accessibilityRole="button" disabled={disabled} onPress={onPress} style={({ pressed }) => [styles.button, (pressed || disabled) && styles.buttonDim]}><Text style={styles.buttonText}>{label}</Text></Pressable>; }
function ErrorText({ value }: { value: string }) { return value ? <Text accessibilityRole="alert" style={styles.error}>{value}</Text> : null; }

const styles = StyleSheet.create({
  flex: { flex: 1 }, safe: { flex: 1, backgroundColor: "#F7F3EC" }, center: { alignItems: "center", justifyContent: "center", gap: 14 }, screen: { flexGrow: 1, justifyContent: "center", padding: 24, gap: 32 }, brand: { marginBottom: 10 }, top: {}, eyebrow: { color: "#6E765C", fontSize: 12, fontWeight: "700", letterSpacing: 2, marginBottom: 12 }, hero: { color: "#292821", fontSize: 34, fontWeight: "600", letterSpacing: -0.8, lineHeight: 40 }, copy: { color: "#68645D", fontSize: 17, lineHeight: 25, marginTop: 12 }, card: { backgroundColor: "#FFFDF9", borderColor: "#E8E0D5", borderRadius: 22, borderWidth: 1, padding: 20, shadowColor: "#332F28", shadowOffset: { width: 0, height: 5 }, shadowOpacity: 0.06, shadowRadius: 18, elevation: 2 }, heading: { color: "#292821", fontSize: 23, fontWeight: "600", marginBottom: 20 }, field: { marginBottom: 17 }, label: { color: "#37342E", fontSize: 14, fontWeight: "600", marginBottom: 7 }, hint: { color: "#817B72", fontSize: 12, marginBottom: 7 }, input: { backgroundColor: "#FAF8F3", borderColor: "#DCD4C8", borderRadius: 12, borderWidth: 1, color: "#292821", fontSize: 16, minHeight: 50, paddingHorizontal: 14, paddingVertical: 12 }, textarea: { minHeight: 112, textAlignVertical: "top" }, inputError: { borderColor: "#A4544B" }, fieldError: { color: "#9C473F", fontSize: 13, marginTop: 6 }, error: { backgroundColor: "#F8EDEA", borderRadius: 10, color: "#8D3F38", fontSize: 14, lineHeight: 20, marginBottom: 14, padding: 11 }, counter: { color: "#817B72", fontSize: 12, marginTop: -10, marginBottom: 14, textAlign: "right" }, button: { alignItems: "center", backgroundColor: "#596447", borderRadius: 13, minHeight: 51, justifyContent: "center", paddingHorizontal: 20 }, buttonDim: { opacity: 0.6 }, buttonText: { color: "#FFFFFF", fontSize: 16, fontWeight: "600" }, link: { color: "#596447", fontSize: 14, fontWeight: "600", marginTop: 18, textAlign: "center" }, profileHeader: { alignItems: "center" }, avatar: { alignItems: "center", backgroundColor: "#DCE2CE", borderRadius: 45, height: 90, justifyContent: "center", marginBottom: 20, width: 90 }, initials: { color: "#465038", fontSize: 29, fontWeight: "600" }, handle: { color: "#6E765C", fontSize: 16, marginTop: 5 }, bio: { color: "#56524B", fontSize: 16, lineHeight: 24, marginTop: 20, textAlign: "center" }, muted: { color: "#817B72", marginTop: 12, textAlign: "center" }, signOut: { color: "#8D3F38", fontSize: 15, fontWeight: "600", marginTop: 20, textAlign: "center" }
});
