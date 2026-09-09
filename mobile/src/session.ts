import { File, Paths } from "expo-file-system";
import { Platform } from "react-native";

export type Session = { accessToken: string; refreshToken: string; expiresAt: number; email?: string };
type Stored = { getItem(key: string): Promise<string | null>; setItem(key: string, value: string): Promise<void>; removeItem(key: string): Promise<void> };

const key = "tribe.supabase.session";
const sessionFile = new File(Paths.document, "tribe-session.json");

const storage: Stored = Platform.OS === "web" ? {
  async getItem(name) { return globalThis.localStorage?.getItem(name) ?? null; },
  async setItem(name, value) { globalThis.localStorage?.setItem(name, value); },
  async removeItem(name) { globalThis.localStorage?.removeItem(name); }
} : {
  async getItem() { return sessionFile.exists ? sessionFile.text() : null; },
  async setItem(_name, value) { sessionFile.write(value); },
  async removeItem() { if (sessionFile.exists) sessionFile.delete(); }
};

type TokenResponse = { access_token: string; refresh_token: string; expires_in: number; user?: { email?: string } };

export class SupabaseSessionClient {
  constructor(private readonly url: string, private readonly publishableKey: string, private readonly store: Stored = storage, private readonly fetcher: typeof fetch = fetch) {}

  async restore(): Promise<Session | null> {
    const value = await this.store.getItem(key);
    if (!value) return null;
    const session = JSON.parse(value) as Session;
    if (session.expiresAt > Date.now() + 60_000) return session;
    try { return await this.refresh(session.refreshToken); } catch { await this.store.removeItem(key); return null; }
  }

  signIn(email: string, password: string) { return this.token("password", { email: email.trim(), password }); }
  signUp(email: string, password: string) { return this.auth("signup", { email: email.trim(), password }); }

  async signOut(session: Session | null): Promise<void> {
    if (session) {
      try { await this.fetcher(`${this.url}/auth/v1/logout`, { method: "POST", headers: this.headers(session.accessToken) }); } catch { /* local sign-out must still succeed */ }
    }
    await this.store.removeItem(key);
  }

  private refresh(refreshToken: string) { return this.token("refresh_token", { refresh_token: refreshToken }); }
  private token(grant: string, body: object) { return this.auth(`token?grant_type=${grant}`, body); }
  private async auth(path: string, body: object): Promise<Session> {
    let response: Response;
    try { response = await this.fetcher(`${this.url}/auth/v1/${path}`, { method: "POST", headers: this.headers(), body: JSON.stringify(body) }); }
    catch { throw new Error("We couldn't reach Tribe. Check your connection and try again."); }
    const result = await response.json() as TokenResponse & { msg?: string; error_description?: string };
    if (!response.ok) throw new Error(result.msg ?? result.error_description ?? "Authentication failed.");
    const session = { accessToken: result.access_token, refreshToken: result.refresh_token, expiresAt: Date.now() + result.expires_in * 1000, email: result.user?.email };
    await this.store.setItem(key, JSON.stringify(session));
    return session;
  }
  private headers(token?: string) { return { apikey: this.publishableKey, "Content-Type": "application/json", ...(token ? { Authorization: `Bearer ${token}` } : {}) }; }
}
