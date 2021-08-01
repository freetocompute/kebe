create sequence public.ssh_keys_id_seq;

CREATE TABLE IF NOT EXISTS public.ssh_keys
(
    id bigint NOT NULL DEFAULT nextval('ssh_keys_id_seq'::regclass),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    public_key_string text COLLATE pg_catalog."default",
    account_id bigint,
    CONSTRAINT ssh_keys_pkey PRIMARY KEY (id),
    CONSTRAINT fk_accounts_ssh_keys FOREIGN KEY (account_id)
        REFERENCES public.accounts (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
)

    TABLESPACE pg_default;

ALTER TABLE public.ssh_keys
    OWNER to manager;

CREATE INDEX idx_ssh_keys_deleted_at
    ON public.ssh_keys USING btree
        (deleted_at ASC NULLS LAST)
    TABLESPACE pg_default;
