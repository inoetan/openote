-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- projects
-- ============================================================
CREATE TABLE IF NOT EXISTS projects (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    created_by  UUID        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_projects_created_by ON projects (created_by);

-- ============================================================
-- users
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(100) NOT NULL UNIQUE,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users (username);
CREATE INDEX IF NOT EXISTS idx_users_email    ON users (email);

-- ============================================================
-- roles  (project-scoped)
-- ============================================================
CREATE TABLE IF NOT EXISTS roles (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID        NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name       VARCHAR(50) NOT NULL,   -- admin | editor | viewer | runner
    UNIQUE (project_id, name)
);

CREATE INDEX IF NOT EXISTS idx_roles_project_id ON roles (project_id);

-- ============================================================
-- user_roles
-- ============================================================
CREATE TABLE IF NOT EXISTS user_roles (
    user_id    UUID NOT NULL REFERENCES users (id)   ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES roles (id)   ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles (user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles (role_id);

-- ============================================================
-- node_definitions
-- ============================================================
CREATE TABLE IF NOT EXISTS node_definitions (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID        NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    type        VARCHAR(50)  NOT NULL,  -- shell_script | shell_command | http_request
    description TEXT        NOT NULL DEFAULT '',
    config      JSONB       NOT NULL DEFAULT '{}',
    created_by  UUID        NOT NULL REFERENCES users (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_node_definitions_project_id ON node_definitions (project_id);
CREATE INDEX IF NOT EXISTS idx_node_definitions_type       ON node_definitions (type);

-- ============================================================
-- jobs
-- ============================================================
CREATE TABLE IF NOT EXISTS jobs (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id     UUID        NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name           VARCHAR(255) NOT NULL,
    description    TEXT        NOT NULL DEFAULT '',
    is_active      BOOLEAN     NOT NULL DEFAULT TRUE,
    timeout_secs   INTEGER     NOT NULL DEFAULT 0,
    max_concurrent INTEGER     NOT NULL DEFAULT 1,
    on_failure     VARCHAR(20) NOT NULL DEFAULT 'stop',  -- stop | continue | retry
    retry_count    INTEGER     NOT NULL DEFAULT 0,
    workflow_spec  JSONB       NOT NULL DEFAULT '{}',
    created_by     UUID        NOT NULL REFERENCES users (id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_jobs_project_id ON jobs (project_id);
CREATE INDEX IF NOT EXISTS idx_jobs_is_active  ON jobs (is_active);

-- ============================================================
-- job_steps  (materialized from workflow_spec)
-- ============================================================
CREATE TABLE IF NOT EXISTS job_steps (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id      UUID        NOT NULL REFERENCES jobs (id) ON DELETE CASCADE,
    node_def_id UUID        NOT NULL REFERENCES node_definitions (id),
    step_order  INTEGER     NOT NULL,
    depends_on  UUID[]      NOT NULL DEFAULT '{}',
    overrides   JSONB       NOT NULL DEFAULT '{}',
    label       TEXT        NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_job_steps_job_id     ON job_steps (job_id);
CREATE INDEX IF NOT EXISTS idx_job_steps_step_order ON job_steps (job_id, step_order);

-- ============================================================
-- schedules
-- ============================================================
CREATE TABLE IF NOT EXISTS schedules (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id      UUID        NOT NULL REFERENCES jobs (id) ON DELETE CASCADE UNIQUE,
    cron_expr   VARCHAR(100) NOT NULL,
    timezone    VARCHAR(100) NOT NULL DEFAULT 'UTC',
    is_enabled  BOOLEAN     NOT NULL DEFAULT TRUE,
    next_run_at TIMESTAMPTZ,
    last_run_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_schedules_job_id    ON schedules (job_id);
CREATE INDEX IF NOT EXISTS idx_schedules_enabled   ON schedules (is_enabled) WHERE is_enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_schedules_next_run  ON schedules (next_run_at) WHERE is_enabled = TRUE;

-- ============================================================
-- executions
-- ============================================================
CREATE TABLE IF NOT EXISTS executions (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id         UUID        NOT NULL REFERENCES jobs (id),
    project_id     UUID        NOT NULL REFERENCES projects (id),
    triggered_by   VARCHAR(20) NOT NULL,  -- schedule | manual | webhook | api
    triggered_user UUID        REFERENCES users (id),
    status         VARCHAR(20) NOT NULL DEFAULT 'queued', -- queued | running | success | failed | aborted | timed_out
    started_at     TIMESTAMPTZ,
    finished_at    TIMESTAMPTZ,
    duration_ms    INTEGER,
    worker_id      TEXT        NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_executions_job_id     ON executions (job_id);
CREATE INDEX IF NOT EXISTS idx_executions_project_id ON executions (project_id);
CREATE INDEX IF NOT EXISTS idx_executions_status     ON executions (status);
CREATE INDEX IF NOT EXISTS idx_executions_created_at ON executions (created_at DESC);

-- ============================================================
-- step_executions
-- ============================================================
CREATE TABLE IF NOT EXISTS step_executions (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID        NOT NULL REFERENCES executions (id) ON DELETE CASCADE,
    job_step_id  UUID        NOT NULL REFERENCES job_steps (id),
    step_order   INTEGER     NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'queued',
    started_at   TIMESTAMPTZ,
    finished_at  TIMESTAMPTZ,
    exit_code    INTEGER,
    log_key      TEXT        NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_step_executions_execution_id ON step_executions (execution_id);
CREATE INDEX IF NOT EXISTS idx_step_executions_step_order   ON step_executions (execution_id, step_order);

-- ============================================================
-- execution_logs
-- ============================================================
CREATE TABLE IF NOT EXISTS execution_logs (
    id           BIGSERIAL   PRIMARY KEY,
    execution_id UUID        NOT NULL REFERENCES executions (id) ON DELETE CASCADE,
    sequence     INTEGER     NOT NULL,
    stream       VARCHAR(10) NOT NULL DEFAULT 'stdout',  -- stdout | stderr
    content      TEXT        NOT NULL,
    logged_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_execution_logs_exec_seq ON execution_logs (execution_id, sequence);

-- ============================================================
-- notification_rules
-- ============================================================
CREATE TABLE IF NOT EXISTS notification_rules (
    id      UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id  UUID        NOT NULL REFERENCES jobs (id) ON DELETE CASCADE,
    event   VARCHAR(20) NOT NULL,    -- on_success | on_failure | on_start | on_timeout
    channel VARCHAR(30) NOT NULL,    -- email | slack_webhook | generic_webhook
    config  JSONB       NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_notification_rules_job_id ON notification_rules (job_id);

-- ============================================================
-- audit_events  (append-only)
-- ============================================================
CREATE TABLE IF NOT EXISTS audit_events (
    id            BIGSERIAL   PRIMARY KEY,
    project_id    UUID        REFERENCES projects (id) ON DELETE SET NULL,
    user_id       UUID        REFERENCES users (id)    ON DELETE SET NULL,
    action        VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id   UUID,
    old_value     JSONB,
    new_value     JSONB,
    ip_address    INET,
    user_agent    TEXT        NOT NULL DEFAULT '',
    occurred_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_events_project_id   ON audit_events (project_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_user_id      ON audit_events (user_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_resource      ON audit_events (resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_occurred_at  ON audit_events (occurred_at DESC);
