create table snap_tracks
(
    id            bigserial not null
        constraint snap_tracks_pkey
            primary key,
    created_at    timestamp with time zone,
    updated_at    timestamp with time zone,
    deleted_at    timestamp with time zone,
    name          text,
    snap_entry_id bigint
        constraint fk_snap_tracks_snap_entry
            references snap_entries
);

create index idx_snap_tracks_deleted_at
    on snap_tracks (deleted_at);