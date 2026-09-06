-- The initial users migration has already been applied to the shared
-- development project, so extend it rather than rewriting migration history.
-- No application profile-creation flow exists yet; deploy this before one is
-- introduced so existing rows do not need a username backfill.
alter table public.users
    add column username text not null
        check (char_length(username) between 3 and 30)
        unique;
