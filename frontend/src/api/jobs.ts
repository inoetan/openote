import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from './client'
import type { Job, NodeDefinition, Schedule, WorkflowSpec, OnFailure } from '../types'

// ── Query keys ───────────────────────────────────────────────────────────────

export const jobKeys = {
  all: ['jobs'] as const,
  byProject: (projectId: string) => [...jobKeys.all, 'project', projectId] as const,
  detail: (projectId: string, jobId: string) =>
    [...jobKeys.all, 'detail', projectId, jobId] as const,
}

export const nodeDefKeys = {
  all: ['nodeDefs'] as const,
  byProject: (projectId: string) => [...nodeDefKeys.all, 'project', projectId] as const,
}

export const scheduleKeys = {
  byJob: (projectId: string, jobId: string) => ['schedules', projectId, jobId] as const,
}

// ── Node Definitions ─────────────────────────────────────────────────────────

export function useNodeDefinitions(projectId: string) {
  return useQuery({
    queryKey: nodeDefKeys.byProject(projectId),
    queryFn: async (): Promise<NodeDefinition[]> => {
      const { data } = await apiClient.get(`/projects/${projectId}/nodes`)
      return data
    },
    enabled: Boolean(projectId),
  })
}

export function useCreateNodeDefinition() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (payload: {
      projectId: string
      name: string
      type: NodeDefinition['type']
      description?: string
      config: Record<string, unknown>
    }): Promise<NodeDefinition> => {
      const { projectId, ...body } = payload
      const { data } = await apiClient.post(`/projects/${projectId}/nodes`, body)
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: nodeDefKeys.byProject(variables.projectId) })
    },
  })
}

export function useUpdateNodeDefinition() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (payload: NodeDefinition): Promise<NodeDefinition> => {
      const { data } = await apiClient.put(
        `/projects/${payload.projectId}/nodes/${payload.id}`,
        payload,
      )
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: nodeDefKeys.byProject(variables.projectId) })
    },
  })
}

export function useDeleteNodeDefinition() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ projectId, id }: { projectId: string; id: string }): Promise<void> => {
      await apiClient.delete(`/projects/${projectId}/nodes/${id}`)
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: nodeDefKeys.byProject(variables.projectId) })
    },
  })
}

// ── Jobs ──────────────────────────────────────────────────────────────────────

export function useJobs(projectId: string) {
  return useQuery({
    queryKey: jobKeys.byProject(projectId),
    queryFn: async (): Promise<Job[]> => {
      const { data } = await apiClient.get(`/projects/${projectId}/jobs`)
      return data
    },
    enabled: Boolean(projectId),
  })
}

export function useJob(
  projectId: string,
  jobId: string,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: jobKeys.detail(projectId, jobId),
    queryFn: async (): Promise<Job> => {
      const { data } = await apiClient.get(`/projects/${projectId}/jobs/${jobId}`)
      return data
    },
    enabled: (options?.enabled ?? true) && Boolean(projectId) && Boolean(jobId),
  })
}

export interface CreateJobPayload {
  projectId: string
  name: string
  description?: string
  timeoutSecs?: number
  maxConcurrent?: number
  onFailure?: OnFailure
  retryCount?: number
  workflowSpec: WorkflowSpec
}

export function useCreateJob() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (payload: CreateJobPayload): Promise<Job> => {
      const { projectId, ...body } = payload
      const { data } = await apiClient.post(`/projects/${projectId}/jobs`, body)
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: jobKeys.byProject(variables.projectId) })
    },
  })
}

export interface UpdateJobPayload {
  projectId: string
  jobId: string
  name?: string
  description?: string
  isActive?: boolean
  timeoutSecs?: number
  maxConcurrent?: number
  onFailure?: OnFailure
  retryCount?: number
  workflowSpec?: WorkflowSpec
}

export function useUpdateJob() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (payload: UpdateJobPayload): Promise<Job> => {
      const { projectId, jobId, ...body } = payload
      const { data } = await apiClient.put(`/projects/${projectId}/jobs/${jobId}`, body)
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: jobKeys.byProject(variables.projectId) })
      queryClient.invalidateQueries({
        queryKey: jobKeys.detail(variables.projectId, variables.jobId),
      })
    },
  })
}

export function useDeleteJob() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      projectId,
      jobId,
    }: {
      projectId: string
      jobId: string
    }): Promise<void> => {
      await apiClient.delete(`/projects/${projectId}/jobs/${jobId}`)
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: jobKeys.byProject(variables.projectId) })
    },
  })
}

// ── Schedules ────────────────────────────────────────────────────────────────

export function useSchedule(projectId: string, jobId: string) {
  return useQuery({
    queryKey: scheduleKeys.byJob(projectId, jobId),
    queryFn: async (): Promise<Schedule | null> => {
      try {
        const { data } = await apiClient.get(`/projects/${projectId}/jobs/${jobId}/schedule`)
        return data
      } catch (e: unknown) {
        if ((e as { response?: { status?: number } })?.response?.status === 404) return null
        throw e
      }
    },
    enabled: Boolean(projectId) && Boolean(jobId),
  })
}

export interface UpsertSchedulePayload {
  projectId: string
  jobId: string
  cronExpr: string
  timezone?: string
  isEnabled?: boolean
}

export function useUpsertSchedule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (payload: UpsertSchedulePayload): Promise<Schedule> => {
      const { projectId, jobId, ...body } = payload
      const { data } = await apiClient.put(
        `/projects/${projectId}/jobs/${jobId}/schedule`,
        body,
      )
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: scheduleKeys.byJob(variables.projectId, variables.jobId),
      })
    },
  })
}

export function useDeleteSchedule() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      projectId,
      jobId,
    }: {
      projectId: string
      jobId: string
    }): Promise<void> => {
      await apiClient.delete(`/projects/${projectId}/jobs/${jobId}/schedule`)
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: scheduleKeys.byJob(variables.projectId, variables.jobId),
      })
    },
  })
}
