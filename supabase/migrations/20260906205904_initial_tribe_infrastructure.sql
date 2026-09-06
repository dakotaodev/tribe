-- Supabase owns authentication. The Go API is the only application-data
-- boundary, so application tables deliberately have no client RLS policies.

create table public.users (
    id uuid primary key references auth.users (id) on delete cascade,
    display_name text not null check (char_length(display_name) between 1 and 80),
    bio text not null default '' check (char_length(bio) <= 500),
    avatar_path text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

alter table public.users enable row level security;

revoke all on table public.users from anon, authenticated;

-- Post images remain private. The Go API will later issue authorized signed
-- uploads and downloads after it has checked relationship and post visibility.
insert into storage.buckets (
    id,
    name,
    public,
    file_size_limit,
    allowed_mime_types
)
values (
    'post-images',
    'post-images',
    false,
    10485760,
    array['image/jpeg', 'image/png', 'image/webp']
);
