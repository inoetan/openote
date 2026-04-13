import { useState, useCallback } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Save, Play, ArrowLeft } from 'lucide-react'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { WorkflowCanvas } from '../components/workflow/WorkflowCanvas'
import { useJob, useCreateJob, useUpdateJob } from '../api/jobs'
import { useExecuteJob } from '../api/executions'
import type { WorkflowSpec, OnFailure } from '../types'

const schema = z.object({
  name: z.string().min(1, 'ジョブ名は必須です'),
  description: z.string().optional(),
  timeoutSecs: z.coerce.number().int().positive().default(3600),
  maxConcurrent: z.coerce.number().int().min(1).default(1),
  onFailure: z.enum(['stop', 'continue', 'retry']).default('stop'),
  retryCount: z.coerce.number().int().min(0).default(0),
})
type FormValues = z.infer<typeof schema>

export default function JobBuilderPage() {
  const { pid, jid } = useParams<{ pid: string; jid?: string }>()
  const navigate = useNavigate()
  const isEditing = !!jid

  const { data: existingJob } = useJob(pid ?? '', jid ?? '', { enabled: isEditing })
  const createJob = useCreateJob()
  const updateJob = useUpdateJob()
  const executeJob = useExecuteJob()

  const [workflowSpec, setWorkflowSpec] = useState<WorkflowSpec | undefined>(
    existingJob?.workflowSpec,
  )
  const [saveError, setSaveError] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    values: existingJob
      ? {
          name: existingJob.name,
          description: existingJob.description ?? '',
          timeoutSecs: existingJob.timeoutSecs,
          maxConcurrent: existingJob.maxConcurrent,
          onFailure: existingJob.onFailure as OnFailure,
          retryCount: existingJob.retryCount,
        }
      : undefined,
  })

  const onWorkflowChange = useCallback((spec: WorkflowSpec) => {
    setWorkflowSpec(spec)
  }, [])

  async function onSubmit(values: FormValues) {
    setSaveError(null)
    if (!workflowSpec || workflowSpec.nodes.length === 0) {
      setSaveError('少なくとも1つのノードをキャンバスに追加してください')
      return
    }
    try {
      const payload = { ...values, workflowSpec }
      if (isEditing && jid) {
        await updateJob.mutateAsync({ projectId: pid!, jobId: jid, ...payload })
      } else {
        await createJob.mutateAsync({ projectId: pid!, ...payload })
      }
      navigate(`/dashboard`)
    } catch (e: unknown) {
      setSaveError('保存に失敗しました')
    }
  }

  async function handleRunNow() {
    if (!jid || !pid) return
    const exec = await executeJob.mutateAsync({ projectId: pid, jobId: jid })
    navigate(`/projects/${pid}/executions?execId=${exec.id}`)
  }

  return (
    <div className="flex h-screen flex-col bg-slate-50">
      {/* Top bar */}
      <header className="flex items-center gap-4 border-b bg-white px-4 py-3 flex-shrink-0">
        <button
          onClick={() => navigate('/dashboard')}
          className="rounded p-1 hover:bg-slate-100"
        >
          <ArrowLeft className="h-4 w-4 text-slate-500" />
        </button>

        <form
          id="job-form"
          onSubmit={handleSubmit(onSubmit)}
          className="flex flex-1 items-center gap-3 flex-wrap"
        >
          <div className="flex flex-col min-w-[200px]">
            <Input
              {...register('name')}
              placeholder="ジョブ名"
              className="text-sm font-medium h-8"
            />
            {errors.name && (
              <p className="text-xs text-red-500">{errors.name.message}</p>
            )}
          </div>

          <Input
            {...register('description')}
            placeholder="説明（任意）"
            className="h-8 text-sm max-w-[240px]"
          />

          <div className="flex items-center gap-1.5">
            <label className="text-xs text-slate-500">タイムアウト(s)</label>
            <Input
              type="number"
              {...register('timeoutSecs')}
              className="h-8 w-20 text-xs"
            />
          </div>

          <div className="flex items-center gap-1.5">
            <label className="text-xs text-slate-500">同時実行</label>
            <Input
              type="number"
              {...register('maxConcurrent')}
              className="h-8 w-16 text-xs"
            />
          </div>

          <div className="flex items-center gap-1.5">
            <label className="text-xs text-slate-500">障害時</label>
            <select {...register('onFailure')} className="h-8 rounded-md border px-2 text-xs">
              <option value="stop">停止</option>
              <option value="continue">続行</option>
              <option value="retry">リトライ</option>
            </select>
          </div>
        </form>

        <div className="flex items-center gap-2">
          {saveError && <p className="text-xs text-red-500">{saveError}</p>}
          {isEditing && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleRunNow}
              disabled={executeJob.isPending}
            >
              <Play className="h-3.5 w-3.5 mr-1 text-green-600" />
              今すぐ実行
            </Button>
          )}
          <Button
            type="submit"
            form="job-form"
            size="sm"
            disabled={isSubmitting}
          >
            <Save className="h-3.5 w-3.5 mr-1" />
            {isSubmitting ? '保存中...' : '保存'}
          </Button>
        </div>
      </header>

      {/* Canvas (takes remaining height) */}
      <div className="flex-1 overflow-hidden">
        <WorkflowCanvas
          initialSpec={existingJob?.workflowSpec ?? workflowSpec}
          onChange={onWorkflowChange}
        />
      </div>
    </div>
  )
}
