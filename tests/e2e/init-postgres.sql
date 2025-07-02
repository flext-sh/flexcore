-- FlexCore Test Database Initialization
-- Creates test data for E2E testing

-- Create test tables
CREATE TABLE IF NOT EXISTS source_data (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(150),
    age INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB,
    active BOOLEAN DEFAULT true
);

CREATE TABLE IF NOT EXISTS processed_data (
    id SERIAL PRIMARY KEY,
    source_id INTEGER REFERENCES source_data(id),
    processed_name VARCHAR(100),
    processed_email VARCHAR(150),
    processing_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    transformation_applied VARCHAR(50),
    status VARCHAR(20) DEFAULT 'pending'
);

CREATE TABLE IF NOT EXISTS api_logs (
    id SERIAL PRIMARY KEY,
    endpoint VARCHAR(255),
    method VARCHAR(10),
    status_code INTEGER,
    response_time_ms INTEGER,
    request_payload JSONB,
    response_payload JSONB,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert test data
INSERT INTO source_data (name, email, age, metadata, active) VALUES
('John Doe', 'john.doe@example.com', 30, '{"department": "engineering", "level": "senior"}', true),
('Jane Smith', 'jane.smith@example.com', 25, '{"department": "marketing", "level": "junior"}', true),
('Bob Johnson', 'bob.johnson@example.com', 35, '{"department": "engineering", "level": "principal"}', true),
('Alice Wilson', 'alice.wilson@example.com', 28, '{"department": "design", "level": "mid"}', true),
('Charlie Brown', 'charlie.brown@example.com', 32, '{"department": "sales", "level": "senior"}', false),
('Diana Prince', 'diana.prince@example.com', 29, '{"department": "hr", "level": "mid"}', true),
('Frank Miller', 'frank.miller@example.com', 41, '{"department": "engineering", "level": "staff"}', true),
('Grace Lee', 'grace.lee@example.com', 26, '{"department": "marketing", "level": "junior"}', true),
('Henry Ford', 'henry.ford@example.com', 38, '{"department": "operations", "level": "senior"}', true),
('Ivy Chen', 'ivy.chen@example.com', 27, '{"department": "design", "level": "mid"}', true);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_source_data_active ON source_data(active);
CREATE INDEX IF NOT EXISTS idx_source_data_created_at ON source_data(created_at);
CREATE INDEX IF NOT EXISTS idx_processed_data_status ON processed_data(status);
CREATE INDEX IF NOT EXISTS idx_api_logs_timestamp ON api_logs(timestamp);

-- Create views for testing
CREATE OR REPLACE VIEW active_users AS
SELECT id, name, email, age, metadata
FROM source_data
WHERE active = true;

CREATE OR REPLACE VIEW department_summary AS
SELECT 
    metadata->>'department' as department,
    COUNT(*) as user_count,
    AVG(age) as avg_age
FROM source_data
WHERE active = true
GROUP BY metadata->>'department';

-- Create functions for testing
CREATE OR REPLACE FUNCTION get_user_by_department(dept_name TEXT)
RETURNS TABLE(id INT, name VARCHAR, email VARCHAR, age INT) AS $$
BEGIN
    RETURN QUERY
    SELECT s.id, s.name, s.email, s.age
    FROM source_data s
    WHERE s.metadata->>'department' = dept_name
    AND s.active = true;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO testuser;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO testuser;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO testuser;

-- Insert more test data for load testing
INSERT INTO source_data (name, email, age, metadata, active)
SELECT 
    'Test User ' || generate_series,
    'test' || generate_series || '@example.com',
    20 + (generate_series % 40),
    jsonb_build_object(
        'department', 
        CASE (generate_series % 5)
            WHEN 0 THEN 'engineering'
            WHEN 1 THEN 'marketing'
            WHEN 2 THEN 'design'
            WHEN 3 THEN 'sales'
            ELSE 'operations'
        END,
        'level',
        CASE (generate_series % 3)
            WHEN 0 THEN 'junior'
            WHEN 1 THEN 'mid'
            ELSE 'senior'
        END,
        'test_data', true
    ),
    (generate_series % 10) != 0
FROM generate_series(1, 1000);

-- Create a table for testing incremental sync
CREATE TABLE IF NOT EXISTS incremental_test (
    id SERIAL PRIMARY KEY,
    data_value TEXT,
    last_modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    version INTEGER DEFAULT 1
);

-- Trigger to update last_modified
CREATE OR REPLACE FUNCTION update_last_modified()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_modified = CURRENT_TIMESTAMP;
    NEW.version = OLD.version + 1;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_incremental_test_modtime
    BEFORE UPDATE ON incremental_test
    FOR EACH ROW
    EXECUTE FUNCTION update_last_modified();

-- Insert initial incremental data
INSERT INTO incremental_test (data_value) VALUES
('Initial data 1'),
('Initial data 2'),
('Initial data 3');

COMMIT;