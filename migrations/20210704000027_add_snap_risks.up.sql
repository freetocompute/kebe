create table snap_risks
(
    id            bigserial not null
        constraint snap_risks_pkey
            primary key,
    created_at    timestamp with time zone,
    updated_at    timestamp with time zone,
    deleted_at    timestamp with time zone,
    name          text,
    snap_track_id bigint
        constraint fk_snap_tracks_risks
            references snap_tracks,
    snap_entry_id bigint
        constraint fk_snap_risks_snap_entry
            references snap_entries,
    revision_id   bigint
        constraint fk_snap_risks_revision
            references snap_revisions
);

create index idx_snap_risks_deleted_at
    on snap_risks (deleted_at);

