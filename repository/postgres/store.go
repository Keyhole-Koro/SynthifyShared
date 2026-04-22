package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/Keyhole-Koro/SynthifyShared/repository"
	"github.com/Keyhole-Koro/SynthifyShared/repository/postgres/sqlcgen"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/oklog/ulid/v2"
)

type Store struct {
	db                 *sql.DB
	queries            *sqlcgen.Queries
	uploadURLGenerator repository.UploadURLGenerator
}

func NewStore(ctx context.Context, dsn string, uploadURLGenerator repository.UploadURLGenerator) (*Store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	store := &Store{db: db, queries: sqlcgen.New(db), uploadURLGenerator: uploadURLGenerator}
	if err := store.ensureSchema(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) q() *sqlcgen.Queries {
	if s.queries == nil {
		s.queries = sqlcgen.New(s.db)
	}
	return s.queries
}

func newID() string {
	return ulid.Make().String()
}

func nowTime() time.Time {
	return time.Now().UTC()
}

func (s *Store) ensureSchema(ctx context.Context) error {
	statements := []string{
		`ALTER TABLE nodes ADD COLUMN IF NOT EXISTS level INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE nodes ADD COLUMN IF NOT EXISTS entity_type TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE nodes ADD COLUMN IF NOT EXISTS governance_state TEXT NOT NULL DEFAULT 'system_generated'`,
		`ALTER TABLE nodes ADD COLUMN IF NOT EXISTS locked_by TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE nodes ADD COLUMN IF NOT EXISTS locked_at TIMESTAMPTZ`,
		`ALTER TABLE nodes ADD COLUMN IF NOT EXISTS last_mutation_job_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE document_processing_jobs ADD COLUMN IF NOT EXISTS requested_by TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE document_processing_jobs ADD COLUMN IF NOT EXISTS capability_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE document_processing_jobs ADD COLUMN IF NOT EXISTS execution_plan_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE document_processing_jobs ADD COLUMN IF NOT EXISTS plan_status TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE document_processing_jobs ADD COLUMN IF NOT EXISTS evaluation_status TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE document_processing_jobs ADD COLUMN IF NOT EXISTS retry_count INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE document_processing_jobs ADD COLUMN IF NOT EXISTS budget_json TEXT NOT NULL DEFAULT '{}'`,
		`CREATE TABLE IF NOT EXISTS document_chunks (
			chunk_id TEXT PRIMARY KEY,
			document_id TEXT NOT NULL REFERENCES documents(document_id) ON DELETE CASCADE,
			heading TEXT NOT NULL DEFAULT '',
			text TEXT NOT NULL,
			source_page INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS job_capabilities (
			capability_id TEXT PRIMARY KEY,
			job_id TEXT NOT NULL REFERENCES document_processing_jobs(job_id) ON DELETE CASCADE,
			workspace_id TEXT NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
			graph_id TEXT NOT NULL REFERENCES graphs(graph_id) ON DELETE CASCADE,
			allowed_document_ids_json TEXT NOT NULL DEFAULT '[]',
			allowed_node_ids_json TEXT NOT NULL DEFAULT '[]',
			allowed_operations_json TEXT NOT NULL DEFAULT '[]',
			max_llm_calls INTEGER NOT NULL DEFAULT 0,
			max_tool_runs INTEGER NOT NULL DEFAULT 0,
			max_node_creations INTEGER NOT NULL DEFAULT 0,
			max_edge_mutations INTEGER NOT NULL DEFAULT 0,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS job_execution_plans (
			plan_id TEXT PRIMARY KEY,
			job_id TEXT NOT NULL REFERENCES document_processing_jobs(job_id) ON DELETE CASCADE,
			status TEXT NOT NULL,
			summary TEXT NOT NULL DEFAULT '',
			plan_json TEXT NOT NULL,
			created_by TEXT NOT NULL DEFAULT 'planner',
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS job_mutation_logs (
			mutation_id TEXT PRIMARY KEY,
			job_id TEXT NOT NULL REFERENCES document_processing_jobs(job_id) ON DELETE CASCADE,
			plan_id TEXT NOT NULL DEFAULT '',
			capability_id TEXT NOT NULL DEFAULT '',
			graph_id TEXT NOT NULL REFERENCES graphs(graph_id) ON DELETE CASCADE,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			mutation_type TEXT NOT NULL,
			risk_tier TEXT NOT NULL DEFAULT '',
			before_json TEXT NOT NULL DEFAULT '{}',
			after_json TEXT NOT NULL DEFAULT '{}',
			provenance_json TEXT NOT NULL DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS job_approval_requests (
			approval_id TEXT PRIMARY KEY,
			job_id TEXT NOT NULL REFERENCES document_processing_jobs(job_id) ON DELETE CASCADE,
			plan_id TEXT NOT NULL REFERENCES job_execution_plans(plan_id) ON DELETE CASCADE,
			status TEXT NOT NULL,
			requested_operations_json TEXT NOT NULL DEFAULT '[]',
			reason TEXT NOT NULL DEFAULT '',
			risk_tier TEXT NOT NULL DEFAULT '',
			requested_by TEXT NOT NULL DEFAULT 'governor',
			reviewed_by TEXT NOT NULL DEFAULT '',
			requested_at TIMESTAMPTZ NOT NULL,
			reviewed_at TIMESTAMPTZ
		)`,
		`CREATE INDEX IF NOT EXISTS idx_document_chunks_document_id ON document_chunks(document_id)`,
		`CREATE INDEX IF NOT EXISTS idx_job_capabilities_job_id ON job_capabilities(job_id)`,
		`CREATE INDEX IF NOT EXISTS idx_job_execution_plans_job_id ON job_execution_plans(job_id)`,
		`CREATE INDEX IF NOT EXISTS idx_job_mutation_logs_job_id ON job_mutation_logs(job_id)`,
	}
	for _, stmt := range statements {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}
