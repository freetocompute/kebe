create sequence public.snap_revisions_id_seq;

CREATE TABLE IF NOT EXISTS public.snap_revisions
(
    id bigint NOT NULL DEFAULT nextval('snap_revisions_id_seq'::regclass),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    snap_filename text COLLATE pg_catalog."default",
    build_assertion_filename text COLLATE pg_catalog."default",
    snap_entry_id bigint,
    sha3_384 text COLLATE pg_catalog."default",
    size bigint,
    CONSTRAINT snap_revisions_pkey PRIMARY KEY (id),
    CONSTRAINT fk_snap_entries_revisions FOREIGN KEY (snap_entry_id)
        REFERENCES public.snap_entries (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
)

TABLESPACE pg_default;

ALTER TABLE public.snap_revisions
    OWNER to manager;
-- Index: idx_snap_revisions_deleted_at

-- DROP INDEX public.idx_snap_revisions_deleted_at;

CREATE INDEX idx_snap_revisions_deleted_at
    ON public.snap_revisions USING btree
    (deleted_at ASC NULLS LAST)
    TABLESPACE pg_default;
