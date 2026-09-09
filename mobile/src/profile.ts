export const profileLimits = {
  usernameMin: 3,
  usernameMax: 30,
  displayNameMax: 80,
  bioMax: 500
} as const;

export type Profile = {
  username: string;
  displayName: string;
  bio: string;
  avatarUrl: string | null;
  createdAt?: string;
  updatedAt?: string;
};

export type ProfileInput = Pick<Profile, "username" | "displayName" | "bio">;
export type ProfileErrors = Partial<Record<keyof ProfileInput, string>>;

export function normalizeProfileInput(input: ProfileInput): ProfileInput {
  return {
    username: input.username.trim().toLowerCase(),
    displayName: input.displayName.trim(),
    bio: input.bio.trim()
  };
}

export function validateProfile(input: ProfileInput): ProfileErrors {
  const normalized = normalizeProfileInput(input);
  const errors: ProfileErrors = {};
  if (normalized.username.length < profileLimits.usernameMin || normalized.username.length > profileLimits.usernameMax) {
    errors.username = "Username must be 3–30 characters.";
  } else if (!/^[a-z0-9_]+$/.test(normalized.username)) {
    errors.username = "Use only lowercase letters, numbers, and underscores.";
  }
  if (!normalized.displayName) errors.displayName = "Display name is required.";
  else if (normalized.displayName.length > profileLimits.displayNameMax) errors.displayName = "Display name must be 80 characters or fewer.";
  if (normalized.bio.length > profileLimits.bioMax) errors.bio = "Bio must be 500 characters or fewer.";
  return errors;
}

export function profileFromWire(value: unknown): Profile {
  const root = value as Record<string, unknown>;
  const raw = ((root.profile as Record<string, unknown> | undefined) ?? root);
  return {
    username: String(raw.username ?? ""),
    displayName: String(raw.display_name ?? raw.displayName ?? ""),
    bio: String(raw.bio ?? ""),
    avatarUrl: (raw.avatar_url ?? raw.avatarUrl ?? null) as string | null,
    createdAt: (raw.created_at ?? raw.createdAt) as string | undefined,
    updatedAt: (raw.updated_at ?? raw.updatedAt) as string | undefined
  };
}
