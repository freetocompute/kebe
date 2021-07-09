alter table snap_revisions
    drop column sha3_384_encoded;

alter table snap_revisions
    add build_assertion_filename text COLLATE pg_catalog."default";
