-- Matches backend/config.yaml defaults (user kaixin). Runs only on first DB init (empty volume).
-- If volume already exists, run once manually:
--   docker exec -it krasis-postgres-dev psql -U krasis -d krasis -c "CREATE USER kaixin WITH SUPERUSER PASSWORD 'kaixin100';"

DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'kaixin') THEN
    CREATE ROLE kaixin LOGIN SUPERUSER PASSWORD 'kaixin100';
  END IF;
END
$$;
