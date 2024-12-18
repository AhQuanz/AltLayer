#!/bin/bash

echo "Initializing the database..."

# Run your SQL script
mysql -u root -p ${MYSQL_ROOT_PASSWORD} ${MYSQL_DATABASE} < /docker-entrypoint-initdb.d/init_db.sql

echo "Database initialized!"
