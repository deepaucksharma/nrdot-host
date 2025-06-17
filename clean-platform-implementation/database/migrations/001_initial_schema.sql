-- Initial database schema for platform
-- Version: 001
-- Date: 2024-01-15

-- Create schema
CREATE SCHEMA IF NOT EXISTS platform;

-- Set search path
SET search_path TO platform, public;

-- Create extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Data points table for raw data storage
CREATE TABLE data_points (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    data_id VARCHAR(255) NOT NULL,
    source VARCHAR(100) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    value NUMERIC,
    metrics JSONB,
    tags JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Indexes for performance
    INDEX idx_data_points_data_id (data_id),
    INDEX idx_data_points_source (source),
    INDEX idx_data_points_timestamp (timestamp DESC),
    INDEX idx_data_points_tags USING GIN (tags),
    INDEX idx_data_points_created_at (created_at DESC)
);

-- Partition by month for better performance
CREATE TABLE data_points_2024_01 PARTITION OF data_points
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE data_points_2024_02 PARTITION OF data_points
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
-- Add more partitions as needed

-- Processed results table
CREATE TABLE processed_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    data_id VARCHAR(255) NOT NULL,
    processing_type VARCHAR(50) NOT NULL,
    parameters JSONB,
    result JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Indexes
    INDEX idx_processed_results_data_id (data_id),
    INDEX idx_processed_results_type (processing_type),
    INDEX idx_processed_results_status (status),
    INDEX idx_processed_results_created_at (created_at DESC)
);

-- Processing queue table
CREATE TABLE processing_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    data_id VARCHAR(255) NOT NULL,
    processing_type VARCHAR(50) NOT NULL,
    parameters JSONB,
    priority INTEGER DEFAULT 5,
    status VARCHAR(20) NOT NULL DEFAULT 'queued',
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    next_retry_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Indexes
    INDEX idx_processing_queue_status_priority (status, priority DESC, created_at),
    INDEX idx_processing_queue_next_retry (next_retry_at)
);

-- API keys table for authentication
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    permissions JSONB DEFAULT '{}',
    rate_limit INTEGER DEFAULT 1000,
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Indexes
    INDEX idx_api_keys_key_hash (key_hash),
    INDEX idx_api_keys_is_active (is_active)
);

-- Audit log table
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    actor_id VARCHAR(255),
    actor_type VARCHAR(50),
    changes JSONB,
    metadata JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Indexes
    INDEX idx_audit_logs_entity (entity_type, entity_id),
    INDEX idx_audit_logs_actor (actor_type, actor_id),
    INDEX idx_audit_logs_created_at (created_at DESC)
);

-- System metrics table
CREATE TABLE system_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    metric_name VARCHAR(100) NOT NULL,
    metric_value NUMERIC NOT NULL,
    labels JSONB DEFAULT '{}',
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Indexes
    INDEX idx_system_metrics_name_timestamp (metric_name, timestamp DESC),
    INDEX idx_system_metrics_labels USING GIN (labels)
);

-- Create hypertable for time-series data (if using TimescaleDB)
-- SELECT create_hypertable('data_points', 'timestamp');
-- SELECT create_hypertable('system_metrics', 'timestamp');

-- Functions and triggers

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_processed_results_updated_at BEFORE UPDATE
    ON processed_results FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_processing_queue_updated_at BEFORE UPDATE
    ON processing_queue FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE
    ON api_keys FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to clean old data
CREATE OR REPLACE FUNCTION clean_old_data(days_to_keep INTEGER DEFAULT 30)
RETURNS void AS $$
BEGIN
    DELETE FROM data_points WHERE created_at < NOW() - INTERVAL '1 day' * days_to_keep;
    DELETE FROM processed_results WHERE created_at < NOW() - INTERVAL '1 day' * days_to_keep;
    DELETE FROM audit_logs WHERE created_at < NOW() - INTERVAL '1 day' * (days_to_keep * 3);
    DELETE FROM system_metrics WHERE timestamp < NOW() - INTERVAL '1 day' * 7;
END;
$$ language 'plpgsql';

-- Materialized view for aggregated metrics
CREATE MATERIALIZED VIEW hourly_metrics AS
SELECT 
    date_trunc('hour', timestamp) as hour,
    source,
    tags->>'environment' as environment,
    COUNT(*) as data_point_count,
    AVG(value) as avg_value,
    MIN(value) as min_value,
    MAX(value) as max_value,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY value) as median_value,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY value) as p95_value,
    PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY value) as p99_value
FROM data_points
WHERE timestamp >= NOW() - INTERVAL '7 days'
GROUP BY hour, source, environment;

CREATE INDEX idx_hourly_metrics_hour ON hourly_metrics(hour DESC);
CREATE INDEX idx_hourly_metrics_source ON hourly_metrics(source);

-- Permissions
GRANT ALL ON SCHEMA platform TO platform_app;
GRANT ALL ON ALL TABLES IN SCHEMA platform TO platform_app;
GRANT ALL ON ALL SEQUENCES IN SCHEMA platform TO platform_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA platform TO platform_app;

-- Read-only user for analytics
CREATE ROLE platform_readonly;
GRANT USAGE ON SCHEMA platform TO platform_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA platform TO platform_readonly;

-- Comments
COMMENT ON SCHEMA platform IS 'Platform application schema';
COMMENT ON TABLE data_points IS 'Raw data points collected from various sources';
COMMENT ON TABLE processed_results IS 'Results from data processing jobs';
COMMENT ON TABLE processing_queue IS 'Queue for pending processing jobs';
COMMENT ON TABLE api_keys IS 'API keys for authentication';
COMMENT ON TABLE audit_logs IS 'Audit trail for all system actions';
COMMENT ON TABLE system_metrics IS 'Internal system performance metrics';