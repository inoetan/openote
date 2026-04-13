import { useState } from 'react'
import { format } from 'date-fns'
import { ja } from 'date-fns/locale'
import { Eye } from 'lucide-react'
import { Button } from '../ui/button'
import { ExecutionStatusBadge } from '../ui/badge'
import { useExecutions } from '../../api/executions'
import { ExecutionDetail } from './ExecutionDetail'
import type { ExecutionStatus } from '../../types'

interface ExecutionHistoryProps {
  projectId: string
  initialExecutionId?: string
}

const STATUS_FILTERS: { label: string; value: ExecutionStatus | '' }[] = [
  { label: '全て', value: '' },
  { label: '実行中', value: 'running' },
  { label: '成功', value: 'success' },
  { label: '失敗', value: 'failed' },
  { label: '待機中', value: 'queued' },
  { label: '中断', value: 'aborted' },
]

export function ExecutionHistory({ projectId, initialExecutionId }: ExecutionHistoryProps) {
  const [statusFilter, setStatusFilter] = useState<ExecutionStatus | ''>('')
  const [selectedId, setSelectedId] = useState<string | null>(initialExecutionId ?? null)
  const { data, isLoading } = useExecutions(projectId)
  const executions = data?.data ?? []

  const filtered = statusFilter
    ? executions.filter((e) => e.status === statusFilter)
    : executions

  function formatDuration(ms?: number) {
    if (!ms) return '-'
    if (ms < 1000) return `${ms}ms`
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
    return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`
  }

  if (isLoading) {
    return <div className="p-8 text-center text-sm text-slate-500">読み込み中...</div>
  }

  return (
    <div className="flex gap-4 h-full">
      {/* List */}
      <div className="flex-1 flex flex-col gap-3 min-w-0">
        {/* Status filter */}
        <div className="flex gap-1.5 flex-wrap">
          {STATUS_FILTERS.map((f) => (
            <button
              key={f.value}
              onClick={() => setStatusFilter(f.value)}
              className={`rounded-full px-3 py-1 text-xs font-medium transition-colors ${
                statusFilter === f.value
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
              }`}
            >
              {f.label}
            </button>
          ))}
        </div>

        <div className="overflow-hidden rounded-xl border border-slate-200">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b bg-slate-50 text-left text-xs font-semibold text-slate-500 uppercase tracking-wide">
                <th className="px-4 py-3">ステータス</th>
                <th className="px-4 py-3">トリガー</th>
                <th className="px-4 py-3">開始</th>
                <th className="px-4 py-3">所要時間</th>
                <th className="px-4 py-3 text-right">詳細</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {filtered.length === 0 && (
                <tr>
                  <td colSpan={5} className="py-10 text-center text-slate-400">
                    実行履歴がありません
                  </td>
                </tr>
              )}
              {filtered.map((exec) => (
                <tr
                  key={exec.id}
                  className={`hover:bg-slate-50 transition-colors cursor-pointer ${
                    selectedId === exec.id ? 'bg-slate-50' : ''
                  }`}
                  onClick={() => setSelectedId(exec.id)}
                >
                  <td className="px-4 py-3">
                    <ExecutionStatusBadge status={exec.status} />
                  </td>
                  <td className="px-4 py-3 text-slate-600 capitalize">{exec.triggeredBy}</td>
                  <td className="px-4 py-3 text-slate-500 text-xs">
                    {exec.startedAt
                      ? format(new Date(exec.startedAt), 'MM/dd HH:mm:ss', { locale: ja })
                      : '-'}
                  </td>
                  <td className="px-4 py-3 text-slate-500 font-mono text-xs">
                    {formatDuration(exec.durationMs)}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Button variant="ghost" size="icon">
                      <Eye className="h-4 w-4 text-slate-400" />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Detail panel */}
      {selectedId && (
        <div className="w-[480px] flex-shrink-0">
          <ExecutionDetail executionId={selectedId} onClose={() => setSelectedId(null)} />
        </div>
      )}
    </div>
  )
}
