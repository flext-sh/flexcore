-- FlexCore PostgreSQL Database Initialization
-- Create users and databases for FlexCore system

-- Create flexcore user if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'flexcore') THEN
        CREATE USER flexcore WITH PASSWORD 'flexcore123';
    END IF;
END
$$;

-- Grant privileges
ALTER USER flexcore CREATEDB;
GRANT ALL PRIVILEGES ON DATABASE flexcore TO flexcore;

-- Create windmill database for workflow engine
SELECT 'CREATE DATABASE windmill OWNER flexcore'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'windmill');

-- Create observability database for metrics
SELECT 'CREATE DATABASE observability OWNER flexcore'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'observability');

-- Grant all privileges
GRANT ALL PRIVILEGES ON DATABASE windmill TO flexcore;
GRANT ALL PRIVILEGES ON DATABASE observability TO flexcore;