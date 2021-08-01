create sequence public.snap_entries_id_seq;

CREATE TABLE IF NOT EXISTS public.snap_entries
(
    id bigint NOT NULL DEFAULT nextval('snap_entries_id_seq'::regclass),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text COLLATE pg_catalog."default",
    snap_store_id text COLLATE pg_catalog."default",
    latest_revision_id bigint,
    type text COLLATE pg_catalog."default",
    confinement text COLLATE pg_catalog."default",
    account_id bigint,
    CONSTRAINT snap_entries_pkey PRIMARY KEY (id),
    CONSTRAINT fk_accounts_snap_entries FOREIGN KEY (account_id)
        REFERENCES public.accounts (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
)

TABLESPACE pg_default;

ALTER TABLE public.snap_entries
    OWNER to manager;
-- Index: idx_snap_entries_deleted_at

-- DROP INDEX public.idx_snap_entries_deleted_at;

CREATE INDEX idx_snap_entries_deleted_at
    ON public.snap_entries USING btree
    (deleted_at ASC NULLS LAST)
    TABLESPACE pg_default;
