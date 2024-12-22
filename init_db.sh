#!/bin/bash

MARKER_FILE="/docker-entrypoint-initdb.d/init.marker"

if [ ! -f "$MARKER_FILE" ]; then
    echo "Initializing DB with default values"
    mysql -u root -p ${MYSQL_ROOT_PASSWORD} < /docker-entrypoint-initdb.d/init_db.sql
    # Create the marker file after successful initialization
    touch "$MARKER_FILE"
else
    echo "DB already initialized. Skipping initialization."
fi