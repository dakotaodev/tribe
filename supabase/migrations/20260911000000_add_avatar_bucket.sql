-- Avatars are private and are only accessed through the authenticated Go API.
-- Object keys are generated as <authenticated-user-id>/<random-id>.<extension>;
-- clients cannot choose a key or write directly to Storage.
insert into storage.buckets (
    id,
    name,
    public,
    file_size_limit,
    allowed_mime_types
)
values (
    'avatars',
    'avatars',
    false,
    5242880,
    array['image/jpeg', 'image/png', 'image/webp']
);

-- Keep the service-role API as the sole application-data boundary. Supabase
-- Storage's service role bypasses RLS; no anon/authenticated object policies
-- are intentionally created for this bucket.
