import { normalizeProfileInput, profileFromWire, type Profile, type ProfileErrors, type ProfileInput } from "./profile";

export class ApiError extends Error {
  constructor(
    public readonly kind: "unauthorized" | "notFound" | "validation" | "conflict" | "network" | "server",
    message: string,
    public readonly fields: ProfileErrors = {}
  ) { super(message); }
}

type Fetch = typeof fetch;

export class ProfileApi {
  constructor(private readonly baseUrl: string, private readonly fetcher: Fetch = fetch) {}

  async getMe(token: string): Promise<Profile | null> {
    const response = await this.request("GET", token);
    if (response.status === 404) return null;
    if (!response.ok) throw await toApiError(response);
    return profileFromWire(await response.json());
  }

  async updateMe(token: string, input: ProfileInput): Promise<Profile> {
    const normalized = normalizeProfileInput(input);
    const response = await this.request("PUT", token, {
      username: normalized.username,
      display_name: normalized.displayName,
      bio: normalized.bio
    });
    if (!response.ok) throw await toApiError(response);
    return profileFromWire(await response.json());
  }

  private async request(method: string, token: string, body?: object): Promise<Response> {
    try {
      return await this.fetcher(`${this.baseUrl}/me`, {
        method,
        headers: { Authorization: `Bearer ${token}`, ...(body ? { "Content-Type": "application/json" } : {}) },
        body: body ? JSON.stringify(body) : undefined
      });
    } catch {
      throw new ApiError("network", "We couldn't reach Tribe. Check your connection and try again.");
    }
  }
}

async function toApiError(response: Response): Promise<ApiError> {
  let body: Record<string, unknown> = {};
  try { body = await response.json() as Record<string, unknown>; } catch { /* non-JSON upstream error */ }
  const detail = ((body.error as Record<string, unknown> | undefined) ?? body);
  const message = typeof detail.message === "string" ? detail.message : "Something went wrong. Please try again.";
  const fields = ((detail.fields ?? detail.errors) as ProfileErrors | undefined) ?? {};
  if (response.status === 401) return new ApiError("unauthorized", "Your session expired. Please sign in again.");
  if (response.status === 404) return new ApiError("notFound", message);
  if (response.status === 409) return new ApiError("conflict", message || "That username is already taken.", fields.username ? fields : { username: "That username is already taken." });
  if (response.status === 400 || response.status === 422) return new ApiError("validation", message, fields);
  return new ApiError("server", message);
}
