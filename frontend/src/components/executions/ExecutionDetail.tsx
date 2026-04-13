import { X, StopCircle, CheckCircle2, Clock, AlertCircle, Loader2 } from 'lucide-react'
import { Button } from '../ui/button'
import { ExecutionStatusBadge } from '../ui/badge'
import { LogViewer } from '../logs/LogViewer'
import { useExecution, useAbortExecution } from '../../api/executions'
import type { ExecutionStatus, StepExecution } from '../../types'

interface ExecutionDetailProps {
  executionId: string
  onClose: () => void
}

const stepStatusIcon: Record<ExecutionStatus, React.ReactNode> = {
  success: <CheckCircle2 className="h-4 w-4 text-green-500" />,
  failed: <AlertCircle className="h-4 w-4 text-red-500" />,
  running: <Loader2 className="h-4 w-4 text-blue-500 animate-spin" />,
  queued: <Clock className="h-4 w-4 text-yellow-500" />,
  aborted: <StopCircle className="h-4 w-4 text-gray-400" />,
  timed_out: <AlertCircle className="h-4 w-4 text-orange-500" />,
}

export function ExecutionDetail({ executionId, onClose }: ExecutionDetailProps) {
  const { data, isLoading } = useExecution(executionId)
  const abortExecution = useAbortExecution()

  if (isLoading || !data) {
    return (
      <div className="flex h-full items-center justify-center rounded-xl border border-slate-200 bg-white">
        <Loader2 className="h-6 w-6 animate-spin text-slate-400" />
      </div>
    )
  }

  const exec = data
  const steps: StepExecution[] = (exec as any).steps ?? []

  const isRunning = exec.status === 'running' || exec.status === 'queued'

  return (
    <div className="flex h-full flex-col rounded-xl border border-slate-200 bg-white overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between border-b px-4 py-3">
        <div className="flex items-center gap-2">
          <ExecutionStatusBadge status={exec.status} />
          <span className="text-xs text-slate-400 font-mono">{exec.id.slice(0, 8)}…</span>
        </div>
        <div className="flex items-center gap-2">
          {isRunning && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => abortExecution.mutate({ projectId: exec.projectId, executionId: exec.id })}
              disabled={abortExecution.isPending}
            >
              <StopCircle className="h-3.5 w-3.5 mr-1 text-red-500" />
              中断
            </Button>
          )}
          <button onClick={onClose} className="rounded p-1 hover:bg-slate-100">
            <X className="h-4 w-4 text-slate-500" />
          </button>
        </div>
      </div>

      {/* Step progress */}
      {steps.length > 0 && (
        <div className="border-b px-4 py-3">
          <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">
            ステップ進捗
          </p>
          <div className="flex flex-col gap-1.5">
            {steps.map((step) => (
              <div key={step.id} className="flex items-center gap-2">
                <span className="flex-shrink-0">{stepStatusIcon[step.status]}</span>
                <span className="text-xs text-slate-600 flex-1">Step {step.stepOrder}</span>
                {step.exitCode !== null && step.exitCode !== undefined && (
                  <span className={`text-xs font-mono ${step.exitCode === 0 ? 'text-green-600' : 'text-red-500'}`}>
                    exit {step.exitCode}
                  </span>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Log viewer */}
      <div className="flex-1 overflow-hidden p-3">
        <LogViewer executionId={executionId} autoScroll={isRunning} />
      </div>
    </div>
  )
}
