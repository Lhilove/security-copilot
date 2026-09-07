CREATE TABLE remediations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    finding_id UUID NOT NULL REFERENCES findings(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    what TEXT NOT NULL,
    risk TEXT NOT NULL,
    fix TEXT NOT NULL,
    proposed_code TEXT,
    can_auto_fix BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL DEFAULT 'pending',
    -- status values: pending, approved, declined, pr_created, pr_merged, failed
    pr_url TEXT,
    pr_number INTEGER,
    approved_at TIMESTAMPTZ,
    declined_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(finding_id, user_id)
);

CREATE INDEX idx_remediations_finding_id ON remediations(finding_id);
CREATE INDEX idx_remediations_user_id ON remediations(user_id);
CREATE INDEX idx_remediations_status ON remediations(status);