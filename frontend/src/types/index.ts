export interface Project {
  id: string
  name: string
  description: string
  createdAt: string
}

export type NodeType = 'shell_script' | 'shell_command' | 'http_request'

export interface NodeDefinition {
  id: string
  projectId: string
  name: string
  type: NodeType
  description: string
  config: Record<string, unknown>
}

export interface WorkflowNode {
  id: string
  type: string
  position: { x: number; y: number }
  data: Record<string, unknown>
}

export interface WorkflowEdge {
  id: string
  source: string
  target: string
  type?: string
  label?: string
}

export interface WorkflowSpec {
  nodes: WorkflowNode[]
  edges: WorkflowEdge[]
  viewport?: Record<string, unknown>
}

export type OnFailure = 'stop' | 'continue' | 'retry'

export interface Job {
  id: string
  projectId: string
  name: string
  description: string
  isActive: boolean
  timeoutSecs: number
  maxConcurrent: number
  onFailure: OnFailure
  retryCount: number
  workflowSpec: WorkflowSpec
}

export type ExecutionStatus =
  | 'queued'
  | 'running'
  | 'success'
  | 'failed'
  | 'aborted'
  | 'timed_out'

export interface Execution {
  id: string
  jobId: string
  projectId: string
  triggeredBy: string
  status: ExecutionStatus
  startedAt?: string
  finishedAt?: string
  durationMs?: number
}

export interface StepExecution {
  id: string
  executionId: string
  jobStepId: string
  stepOrder: number
  status: ExecutionStatus
  startedAt?: string
  finishedAt?: string
  exitCode?: number | null
}

export interface Schedule {
  id: string
  jobId: string
  cronExpr: string
  timezone: string
  isEnabled: boolean
  nextRunAt?: string
  lastRunAt?: string
}

export interface User {
  id: string
  username: string
  email: string
  isActive: boolean
}

export interface LogChunk {
  seq: number
  stream: 'stdout' | 'stderr' | 'system'
  content: string
  ts: string
}

export interface LoginResponse {
  accessToken: string
  refreshToken: string
  user: User
}

export interface ApiError {
  message: string
  code?: string
}
