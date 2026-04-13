-- Drop tables in reverse dependency order to respect foreign key constraints.

DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS notification_rules;
DROP TABLE IF EXISTS execution_logs;
DROP TABLE IF EXISTS step_executions;
DROP TABLE IF EXISTS executions;
DROP TABLE IF EXISTS schedules;
DROP TABLE IF EXISTS job_steps;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS node_definitions;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS projects;
