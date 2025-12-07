-- Initial schema for Kubernetes Monitoring & Alerting System
-- Creates the alerts table for tracking all alert events

CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Alert identification
    status VARCHAR(20) NOT NULL DEFAULT 'firing',

    -- Alert content
    severity VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    source VARCHAR(100) NOT NULL,
    labels JSONB DEFAULT '{}',
    value DOUBLE PRECISION,

    -- Timestamps
    triggered_at TIMESTAMP WITH TIME ZONE NOT NULL,
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for efficient queries
CREATE INDEX idx_alerts_status ON alerts(status);
CREATE INDEX idx_alerts_severity ON alerts(severity);
CREATE INDEX idx_alerts_triggered_at ON alerts(triggered_at DESC);
CREATE INDEX idx_alerts_source ON alerts(source);

-- JSONB index for label searches
CREATE INDEX idx_alerts_labels ON alerts USING GIN(labels);
