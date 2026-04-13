import { useNavigate } from 'react-router-dom'
import { Play, Settings, Clock, Pencil, Trash2 } from 'lucide-react'
import { Button } from '../ui/button'
import { Badge } from '../ui/badge'
import { useJobs, useDeleteJob } from '../../api/jobs'
import { useExecuteJob } from '../../api/executions'
import type { Job } from '../../types'

interface JobListProps {
  projectId: string
}

export function JobList({ projectId }: JobListProps) {
  const navigate = useNavigate()
  const { data: jobs = [], isLoading } = useJobs(projectId)
  const deleteJob = useDeleteJob()
  const executeJob = useExecuteJob()

  async function handleRun(job: Job) {
    const exec = await executeJob.mutateAsync({ projectId, jobId: job.id })
    navigate(`/projects/${projectId}/executions?execId=${exec.id}`)
  }

  async function handleDelete(job: Job) {
    if (!confirm(`「${job.name}」を削除しますか？`)) return
    await deleteJob.mutateAsync({ projectId, jobId: job.id })
  }

  if (isLoading) {
    return <div className="p-8 text-center text-sm text-slate-500">読み込み中...</div>
  }

  if (jobs.length === 0) {
    return (
      <div className="p-12 text-center">
        <p className="text-slate-500 mb-4">ジョブがまだありません</p>
        <Button onClick={() => navigate(`/projects/${projectId}/jobs/new`)}>
          最初のジョブを作成
        </Button>
      </div>
    )
  }

  return (
    <div className="overflow-hidden rounded-xl border border-slate-200">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-slate-50 text-left text-xs font-semibold text-slate-500 uppercase tracking-wide">
            <th className="px-4 py-3">ジョブ名</th>
            <th className="px-4 py-3">状態</th>
            <th className="px-4 py-3">同時実行</th>
            <th className="px-4 py-3">障害時</th>
            <th className="px-4 py-3 text-right">操作</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {jobs.map((job) => (
            <tr key={job.id} className="hover:bg-slate-50 transition-colors">
              <td className="px-4 py-3">
                <div className="font-medium text-slate-800">{job.name}</div>
                {job.description && (
                  <div className="text-xs text-slate-400 mt-0.5">{job.description}</div>
                )}
              </td>
              <td className="px-4 py-3">
                <Badge variant={job.isActive ? 'success' : 'secondary'}>
                  {job.isActive ? '有効' : '無効'}
                </Badge>
              </td>
              <td className="px-4 py-3 text-slate-600">{job.maxConcurrent}</td>
              <td className="px-4 py-3 text-slate-600">{job.onFailure}</td>
              <td className="px-4 py-3">
                <div className="flex items-center justify-end gap-1">
                  <Button
                    variant="ghost"
                    size="icon"
                    title="実行"
                    onClick={() => handleRun(job)}
                    disabled={executeJob.isPending}
                  >
                    <Play className="h-4 w-4 text-green-600" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    title="スケジュール"
                    onClick={() => navigate(`/projects/${projectId}/jobs/${job.id}/schedule`)}
                  >
                    <Clock className="h-4 w-4 text-slate-500" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    title="編集"
                    onClick={() => navigate(`/projects/${projectId}/jobs/${job.id}/edit`)}
                  >
                    <Pencil className="h-4 w-4 text-slate-500" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    title="削除"
                    onClick={() => handleDelete(job)}
                    disabled={deleteJob.isPending}
                  >
                    <Trash2 className="h-4 w-4 text-red-400" />
                  </Button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
