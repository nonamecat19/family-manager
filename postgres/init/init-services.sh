#!/bin/sh
set -e

SERVICE_DATABASES="auth family finance notes recipes"

for db in ${SERVICE_DATABASES}; do
	exists=$(psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB:-postgres}" -tAc \
		"SELECT 1 FROM pg_database WHERE datname = '${db}'")
	if [ "${exists}" = "1" ]; then
		echo "database ${db} already exists, skipping"
		continue
	fi
	echo "creating database ${db} owned by ${POSTGRES_USER}"
	psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB:-postgres}" -v ON_ERROR_STOP=1 \
		-c "CREATE DATABASE \"${db}\" OWNER \"${POSTGRES_USER}\""
done
