import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from './client'
import type { Execution, LogChunk, StepExecution } from '../types'

// ── Query keys ───────────────────────────────────────────────────────────────

export const executionKeys = {
  all: ['executions'] as const,
  byProject: (projectId: string) => [...executionKeys.all, 'project', projectId] as const,
  detail: (executionId: string) => [...executionKeys.all, 'detail', executionId] as const,
  logs: (executionId: string) => [...executionKeys.all, 'logs', executionId] as const,
  steps: (executionId: string) => [...executionKeys.all, 'steps', executionId] as const,
}

export interface PaginatedExecutions {
  data: Execution[]
  total: number
  page: number
  pageSize: number
}

export function useExecutions(projectId: string) {
  return useQuery({
    queryKey: executionKeys.byProject(projectId),
    queryFn: async (): Promise<PaginatedExecutions> => {
      const { data } = await apiClient.get(`/projects/${projectId}/executions`)
      // Handle both paginated and plain array responses
      if (Array.isArray(data)) {
        return { data, total: data.length, page: 1, pageSize: data.length }
      }
      return data
    },
    enabled: Boolean(projectId),
    refetchInterval: 5000,
  })
}

export function useExecution(executionId: string) {
  return useQuery({
    queryKey: executionKeys.detail(executionId),
    queryFn: async (): Promise<Execution & { steps?: StepExecution[] }> => {
      const { data } = await apiClient.get(`/projects/any/executions/${executionId}`)
      return data
    },
    enabled: Boolean(executionId),
    refetchInterval: (query) => {
      const status = query.state.data?.status
      if (
        status === 'success' ||
        status === 'failed' ||
        status === 'aborted' ||
        status === 'timed_out'
      ) {
        return false
      }
      return 3000
    },
  })
}

export function useExecutionLogs(executionId: string, fromSeq = 0) {
  return useQuery({
    queryKey: [...executionKeys.logs(executionId), fromSeq],
    queryFn: async (): Promise<LogChunk[]> => {
      const { data } = await apiClient.get(`/executions/${executionId}/logs`, {
        params: { fromSeq },
      })
      return data
    },
    enabled: Boolean(executionId),
  })
}

export function useExecuteJob() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      projectId,
      jobId,
    }: {
      projectId: string
      jobId: string
    }): Promise<Execution> => {
      const { data } = await apiClient.post(`/projects/${projectId}/jobs/${jobId}/execute`)
      return data
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: executionKeys.byProject(variables.projectId) })
    },
  })
}

export function useAbortExecution() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      projectId,
      executionId,
    }: {
      projectId: string
      executionId: string
    }): Promise<void> => {
      await apiClient.post(`/projects/${projectId}/executions/${executionId}/abort`)
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: executionKeys.detail(variables.executionId) })
    },
  })
}
