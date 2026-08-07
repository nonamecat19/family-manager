-- One database per service: a service owns its tables, and "owns" includes not being able to
-- read another service's by accident. Schemas inside one database would not enforce that.
--
-- Runs only on an empty data volume (docker-entrypoint-initdb.d). To add these to an existing
-- volume: docker compose exec postgres psql -U admin -f /docker-entrypoint-initdb.d/init-services.sql
CREATE DATABASE auth OWNER admin;
CREATE DATABASE family OWNER admin;
CREATE DATABASE finance OWNER admin;
