#!/bin/bash

HOST=$1
POST=$2
pg_dump -h ${HOST}  -p ${PORT} -U manager --column-inserts --data-only  store > /backup/store_inserts.sql
