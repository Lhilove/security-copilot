CREATE TABLE findings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    source_alert_id TEXT NOT NULL,
    severity TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    state TEXT NOT NULL,
    file_path TEXT,
    line_number INTEGER,
    package_name TEXT,
    cve_id TEXT,
    secret_type TEXT,
    raw_data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(repository_id, source, source_alert_id)
);

CREATE INDEX idx_findings_repository_id ON findings(repository_id);
CREATE INDEX idx_findings_source ON findings(source);
CREATE INDEX idx_findings_severity ON findings(severity);
CREATE INDEX idx_findings_state ON findings(state);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_findings_updated_at
    BEFORE UPDATE ON findings
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();