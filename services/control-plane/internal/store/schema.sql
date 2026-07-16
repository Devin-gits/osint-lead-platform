CREATE TABLE IF NOT EXISTS leads (
    id uuid PRIMARY KEY,
    name text,
    email text,
    phone text,
    company text,
    domain text,
    source_id text,
    permission_ref text,
    stage text NOT NULL DEFAULT 'raw',
    risk_level text NOT NULL DEFAULT 'n/a',
    risk_score numeric,
    results jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_leads_stage ON leads(stage);
CREATE INDEX IF NOT EXISTS idx_leads_risk ON leads(risk_level);
CREATE INDEX IF NOT EXISTS idx_leads_email ON leads(email);
CREATE INDEX IF NOT EXISTS idx_leads_created_at ON leads(created_at DESC);

CREATE TABLE IF NOT EXISTS audit_events (
    id uuid PRIMARY KEY,
    lead_id uuid NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    run_id uuid,
    module text NOT NULL,
    tool text NOT NULL,
    checked_at timestamptz NOT NULL,
    status text NOT NULL,
    legal_basis text NOT NULL,
    subject jsonb NOT NULL DEFAULT '{}'::jsonb,
    raw_json jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_lead ON audit_events(lead_id);
CREATE INDEX IF NOT EXISTS idx_audit_module ON audit_events(module);
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_events(created_at DESC);

CREATE TABLE IF NOT EXISTS pipeline_runs (
    id uuid PRIMARY KEY,
    type text NOT NULL,
    status text NOT NULL,
    started_at timestamptz NOT NULL,
    finished_at timestamptz,
    lead_ids jsonb NOT NULL DEFAULT '[]'::jsonb,
    modules_executed jsonb NOT NULL DEFAULT '[]'::jsonb,
    audit_event_ids jsonb NOT NULL DEFAULT '[]'::jsonb,
    legal_basis text NOT NULL,
    permission_refs jsonb NOT NULL DEFAULT '[]'::jsonb,
    error text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_runs_status ON pipeline_runs(status);
CREATE INDEX IF NOT EXISTS idx_runs_started ON pipeline_runs(started_at DESC);
