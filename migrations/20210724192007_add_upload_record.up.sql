create sequence public.snap_uploads_id_seq;

CREATE TABLE IF NOT EXISTS public.snap_uploads
(
    id         bigint NOT NULL DEFAULT nextval('snap_uploads_id_seq'::regclass),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text COLLATE pg_catalog."default",
    up_down_id text COLLATE pg_catalog."default",
    filesize bigint,
    snap_entry_id bigint,
    channels text COLLATE pg_catalog."default",
    CONSTRAINT snap_uploads_pkey PRIMARY KEY (id),
    CONSTRAINT fk_snap_entries_uploads FOREIGN KEY (snap_entry_id)
        REFERENCES public.snap_entries (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
) TABLESPACE pg_default;

ALTER TABLE public.snap_uploads
    OWNER to manager;