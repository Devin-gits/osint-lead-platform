package store

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/models"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed schema.sql
var schemaSQL string

// PostgresStore implements Store using Postgres via pgx/stdlib.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore opens a connection to databaseURL, runs embedded schema migrations,
// and returns a Store. It panics only if the schema cannot be applied.
func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

// Close releases the database pool.
func (p *PostgresStore) Close() error {
	return p.db.Close()
}

func (p *PostgresStore) CreateLead(ctx context.Context, lead models.Lead) (models.Lead, error) {
	if lead.ID == "" {
		return models.Lead{}, ErrInvalid
	}
	if lead.Stage == "" {
		lead.Stage = models.StageRaw
	}
	if lead.RiskLevel == "" {
		lead.RiskLevel = models.RiskUnknown
	}
	if lead.Results == nil {
		lead.Results = map[string]any{}
	}

	resultsJSON, err := json.Marshal(lead.Results)
	if err != nil {
		return models.Lead{}, fmt.Errorf("marshal results: %w", err)
	}

	var rs sql.NullFloat64
	if lead.RiskScore != nil {
		rs.Valid = true
		rs.Float64 = *lead.RiskScore
	}

	now := time.Now().UTC()
	_, err = p.db.ExecContext(ctx, `
		INSERT INTO leads (id, name, email, phone, company, domain, source_id, permission_ref, stage, risk_level, risk_score, results, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $13)
	`, lead.ID, nullString(lead.Name), nullString(lead.Email), nullString(lead.Phone),
		nullString(lead.Company), nullString(lead.Domain), nullString(lead.SourceID),
		nullString(lead.PermissionRef), lead.Stage, lead.RiskLevel, rs, resultsJSON, now)
	if err != nil {
		return models.Lead{}, fmt.Errorf("insert lead: %w", err)
	}

	lead.CreatedAt = now
	lead.UpdatedAt = now
	return lead, nil
}

func (p *PostgresStore) ListLeads(ctx context.Context, params models.LeadSearchParams) ([]models.Lead, int, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT id, name, email, phone, company, domain, source_id, permission_ref, stage, risk_level, risk_score, results, created_at, updated_at
		FROM leads
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, 0, fmt.Errorf("query leads: %w", err)
	}
	defer rows.Close()

	all, err := scanLeads(rows)
	if err != nil {
		return nil, 0, err
	}

	filtered, total := filterLeads(all, params)
	return filtered, total, nil
}

func (p *PostgresStore) GetLead(ctx context.Context, id string) (models.Lead, error) {
	row := p.db.QueryRowContext(ctx, `
		SELECT id, name, email, phone, company, domain, source_id, permission_ref, stage, risk_level, risk_score, results, created_at, updated_at
		FROM leads WHERE id = $1
	`, id)
	return scanLead(row)
}

func (p *PostgresStore) UpdateLead(ctx context.Context, lead models.Lead) (models.Lead, error) {
	resultsJSON, err := json.Marshal(lead.Results)
	if err != nil {
		return models.Lead{}, fmt.Errorf("marshal results: %w", err)
	}

	var rs sql.NullFloat64
	if lead.RiskScore != nil {
		rs.Valid = true
		rs.Float64 = *lead.RiskScore
	}

	now := time.Now().UTC()
	res, err := p.db.ExecContext(ctx, `
		UPDATE leads
		SET name = $1, email = $2, phone = $3, company = $4, domain = $5, source_id = $6,
		    permission_ref = $7, stage = $8, risk_level = $9, risk_score = $10, results = $11, updated_at = $12
		WHERE id = $13
	`, nullString(lead.Name), nullString(lead.Email), nullString(lead.Phone),
		nullString(lead.Company), nullString(lead.Domain), nullString(lead.SourceID),
		nullString(lead.PermissionRef), lead.Stage, lead.RiskLevel, rs, resultsJSON, now, lead.ID)
	if err != nil {
		return models.Lead{}, fmt.Errorf("update lead: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return models.Lead{}, ErrNotFound
	}
	lead.UpdatedAt = now
	return lead, nil
}

func (p *PostgresStore) CreateAuditEvent(ctx context.Context, event models.AuditEvent) (models.AuditEvent, error) {
	if event.ID == "" {
		return models.AuditEvent{}, ErrInvalid
	}
	subjectJSON, err := json.Marshal(event.Subject)
	if err != nil {
		return models.AuditEvent{}, fmt.Errorf("marshal subject: %w", err)
	}

	event.CreatedAt = time.Now().UTC()
	_, err = p.db.ExecContext(ctx, `
		INSERT INTO audit_events (id, lead_id, run_id, module, tool, checked_at, status, legal_basis, subject, raw_stderr_json, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, event.ID, event.LeadID, nullStringPtr(event.RunID), event.Module, event.Tool,
		event.CheckedAt.UTC(), event.Status, event.LegalBasis, subjectJSON, nullString(event.RawStderrJSON), event.CreatedAt)
	if err != nil {
		return models.AuditEvent{}, fmt.Errorf("insert audit event: %w", err)
	}
	return event, nil
}

func (p *PostgresStore) ListAuditEvents(ctx context.Context, params models.AuditSearchParams) ([]models.AuditEvent, int, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT id, lead_id, run_id, module, tool, checked_at, status, legal_basis, subject, raw_stderr_json, created_at
		FROM audit_events
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit events: %w", err)
	}
	defer rows.Close()

	events, err := scanAuditEvents(rows)
	if err != nil {
		return nil, 0, err
	}

	// Apply in-memory filtering so both Store implementations behave identically.
	page, pageSize := normalizePagination(params.Page, params.PageSize)
	filtered := make([]models.AuditEvent, 0, len(events))
	for _, e := range events {
		if params.Module != "" && !strings.EqualFold(e.Module, params.Module) {
			continue
		}
		if params.Status != "" && !strings.EqualFold(e.Status, params.Status) {
			continue
		}
		filtered = append(filtered, e)
	}

	total := len(filtered)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return filtered[start:end], total, nil
}

func (p *PostgresStore) ListAuditEventsByLead(ctx context.Context, leadID string) ([]models.AuditEvent, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT id, lead_id, run_id, module, tool, checked_at, status, legal_basis, subject, raw_stderr_json, created_at
		FROM audit_events
		WHERE lead_id = $1
		ORDER BY checked_at DESC
	`, leadID)
	if err != nil {
		return nil, fmt.Errorf("query audit events by lead: %w", err)
	}
	defer rows.Close()
	return scanAuditEvents(rows)
}

func (p *PostgresStore) CreatePipelineRun(ctx context.Context, run models.PipelineRun) (models.PipelineRun, error) {
	if run.ID == "" {
		return models.PipelineRun{}, ErrInvalid
	}
	run.CreatedAt = time.Now().UTC()
	if err := p.saveRun(ctx, run); err != nil {
		return models.PipelineRun{}, err
	}
	return run, nil
}

func (p *PostgresStore) ListPipelineRuns(ctx context.Context, params models.AuditSearchParams) ([]models.PipelineRun, int, error) {
	page, pageSize := normalizePagination(params.Page, params.PageSize)
	rows, err := p.db.QueryContext(ctx, `
		SELECT id, type, status, started_at, finished_at, lead_ids, modules_executed, audit_event_ids, legal_basis, permission_refs, error, created_at
		FROM pipeline_runs
		ORDER BY started_at DESC
		LIMIT $1 OFFSET $2
	`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("query runs: %w", err)
	}
	defer rows.Close()

	runs, err := scanPipelineRuns(rows)
	if err != nil {
		return nil, 0, err
	}

	var total int
	if err := p.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pipeline_runs`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count runs: %w", err)
	}
	return runs, total, nil
}

func (p *PostgresStore) GetPipelineRun(ctx context.Context, id string) (models.PipelineRun, error) {
	row := p.db.QueryRowContext(ctx, `
		SELECT id, type, status, started_at, finished_at, lead_ids, modules_executed, audit_event_ids, legal_basis, permission_refs, error, created_at
		FROM pipeline_runs WHERE id = $1
	`, id)
	return scanPipelineRun(row)
}

func (p *PostgresStore) UpdatePipelineRun(ctx context.Context, run models.PipelineRun) (models.PipelineRun, error) {
	if err := p.saveRun(ctx, run); err != nil {
		return models.PipelineRun{}, err
	}
	return run, nil
}

func (p *PostgresStore) saveRun(ctx context.Context, run models.PipelineRun) error {
	leadIDs, err := json.Marshal(run.LeadIDs)
	if err != nil {
		return fmt.Errorf("marshal lead_ids: %w", err)
	}
	modules, err := json.Marshal(run.ModulesExecuted)
	if err != nil {
		return fmt.Errorf("marshal modules_executed: %w", err)
	}
	auditIDs, err := json.Marshal(run.AuditEventIDs)
	if err != nil {
		return fmt.Errorf("marshal audit_event_ids: %w", err)
	}
	permRefs, err := json.Marshal(run.PermissionRefs)
	if err != nil {
		return fmt.Errorf("marshal permission_refs: %w", err)
	}

	var finished sql.NullTime
	if run.FinishedAt != nil {
		finished.Valid = true
		finished.Time = *run.FinishedAt
	}

	_, err = p.db.ExecContext(ctx, `
		INSERT INTO pipeline_runs (id, type, status, started_at, finished_at, lead_ids, modules_executed, audit_event_ids, legal_basis, permission_refs, error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
		    status = EXCLUDED.status,
		    finished_at = EXCLUDED.finished_at,
		    lead_ids = EXCLUDED.lead_ids,
		    modules_executed = EXCLUDED.modules_executed,
		    audit_event_ids = EXCLUDED.audit_event_ids,
		    legal_basis = EXCLUDED.legal_basis,
		    permission_refs = EXCLUDED.permission_refs,
		    error = EXCLUDED.error
	`, run.ID, run.Type, run.Status, run.StartedAt.UTC(), finished, leadIDs, modules, auditIDs,
		run.LegalBasis, permRefs, nullString(run.Error), run.CreatedAt.UTC())
	if err != nil {
		return fmt.Errorf("upsert run: %w", err)
	}
	return nil
}

func (p *PostgresStore) ComplianceSummary(ctx context.Context) (models.ComplianceSummary, error) {
	_ = ctx
	return staticComplianceSummary(), nil
}

// Helpers.

type scannable interface {
	Scan(dest ...any) error
}

func scanLead(s scannable) (models.Lead, error) {
	var l models.Lead
	var results []byte
	var rs sql.NullFloat64
	var name, email, phone, company, domain, sourceID, permissionRef sql.NullString
	err := s.Scan(&l.ID, &name, &email, &phone, &company, &domain, &sourceID,
		&permissionRef, &l.Stage, &l.RiskLevel, &rs, &results, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return models.Lead{}, ErrNotFound
		}
		return models.Lead{}, fmt.Errorf("scan lead: %w", err)
	}
	if name.Valid {
		l.Name = name.String
	}
	if email.Valid {
		l.Email = email.String
	}
	if phone.Valid {
		l.Phone = phone.String
	}
	if company.Valid {
		l.Company = company.String
	}
	if domain.Valid {
		l.Domain = domain.String
	}
	if sourceID.Valid {
		l.SourceID = sourceID.String
	}
	if permissionRef.Valid {
		l.PermissionRef = permissionRef.String
	}
	if rs.Valid {
		l.RiskScore = &rs.Float64
	}
	if len(results) > 0 {
		if err := json.Unmarshal(results, &l.Results); err != nil {
			return models.Lead{}, fmt.Errorf("unmarshal results: %w", err)
		}
	} else {
		l.Results = map[string]any{}
	}
	return l, nil
}

func scanLeads(rows *sql.Rows) ([]models.Lead, error) {
	var leads []models.Lead
	for rows.Next() {
		l, err := scanLead(rows)
		if err != nil {
			return nil, err
		}
		leads = append(leads, l)
	}
	return leads, rows.Err()
}

func scanAuditEvent(s scannable) (models.AuditEvent, error) {
	var e models.AuditEvent
	var subject []byte
	var raw sql.NullString
	var runID sql.NullString
	err := s.Scan(&e.ID, &e.LeadID, &runID, &e.Module, &e.Tool, &e.CheckedAt, &e.Status,
		&e.LegalBasis, &subject, &raw, &e.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return models.AuditEvent{}, ErrNotFound
		}
		return models.AuditEvent{}, fmt.Errorf("scan audit event: %w", err)
	}
	if runID.Valid {
		e.RunID = &runID.String
	}
	if len(subject) > 0 {
		_ = json.Unmarshal(subject, &e.Subject)
	}
	if raw.Valid && raw.String != "" && raw.String != "null" {
		e.RawStderrJSON = raw.String
	}
	return e, nil
}

func scanAuditEvents(rows *sql.Rows) ([]models.AuditEvent, error) {
	var events []models.AuditEvent
	for rows.Next() {
		e, err := scanAuditEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func scanPipelineRun(s scannable) (models.PipelineRun, error) {
	var r models.PipelineRun
	var leadIDs, modules, auditIDs, permRefs []byte
	var finished sql.NullTime
	var errStr sql.NullString
	err := s.Scan(&r.ID, &r.Type, &r.Status, &r.StartedAt, &finished, &leadIDs, &modules, &auditIDs,
		&r.LegalBasis, &permRefs, &errStr, &r.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return models.PipelineRun{}, ErrNotFound
		}
		return models.PipelineRun{}, fmt.Errorf("scan run: %w", err)
	}
	if finished.Valid {
		r.FinishedAt = &finished.Time
	}
	if errStr.Valid {
		r.Error = errStr.String
	}
	if err := json.Unmarshal(leadIDs, &r.LeadIDs); err != nil {
		return models.PipelineRun{}, fmt.Errorf("unmarshal lead_ids: %w", err)
	}
	if err := json.Unmarshal(modules, &r.ModulesExecuted); err != nil {
		return models.PipelineRun{}, fmt.Errorf("unmarshal modules: %w", err)
	}
	if err := json.Unmarshal(auditIDs, &r.AuditEventIDs); err != nil {
		return models.PipelineRun{}, fmt.Errorf("unmarshal audit_ids: %w", err)
	}
	if err := json.Unmarshal(permRefs, &r.PermissionRefs); err != nil {
		return models.PipelineRun{}, fmt.Errorf("unmarshal permission_refs: %w", err)
	}
	return r, nil
}

func scanPipelineRuns(rows *sql.Rows) ([]models.PipelineRun, error) {
	var runs []models.PipelineRun
	for rows.Next() {
		r, err := scanPipelineRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func nullStringPtr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}
