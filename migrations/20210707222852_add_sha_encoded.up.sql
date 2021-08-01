alter table snap_revisions
    add sha3_384_encoded text;

alter table snap_revisions drop column build_assertion_filename;

