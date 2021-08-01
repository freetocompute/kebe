create table snap_branches
(
    id            bigserial not null
        constraint snap_branches_pkey
            primary key,
    created_at    timestamp with time zone,
    updated_at    timestamp with time zone,
    deleted_at    timestamp with time zone,
    name          text,
    snap_risk_id  bigint
        constraint fk_snap_risks_branches
            references snap_risks,
    snap_entry_id bigint
        constraint fk_snap_branches_snap_entry
            references snap_entries,
    revision_id   bigint
        constraint fk_snap_branches_revision
            references snap_revisions
);

create index idx_snap_branches_deleted_at
    on snap_branches (deleted_at);
