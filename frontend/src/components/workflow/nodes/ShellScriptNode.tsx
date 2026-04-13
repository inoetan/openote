import { Handle, Position, type NodeProps } from '@xyflow/react'
import { Terminal } from 'lucide-react'
import { cn } from '../../../lib/utils'
import type { ExecutionStatus } from '../../../types'

export interface ShellScriptNodeData {
  label: string
  nodeDefId?: string
  interpreter?: string
  scriptContent?: string
  executionStatus?: ExecutionStatus
  [key: string]: unknown
}

const statusBorder: Partial<Record<ExecutionStatus, string>> = {
  running: 'border-blue-400 shadow-blue-200',
  success: 'border-green-400 shadow-green-200',
  failed: 'border-red-400 shadow-red-200',
  aborted: 'border-gray-400',
}

export function ShellScriptNode({ data, selected }: NodeProps) {
  const d = data as ShellScriptNodeData
  const statusCls = d.executionStatus ? statusBorder[d.executionStatus] ?? '' : ''

  return (
    <div
      className={cn(
        'rounded-lg border-2 bg-white px-4 py-3 shadow-md transition-all min-w-[160px]',
        selected ? 'border-primary ring-2 ring-primary/30' : 'border-slate-300',
        statusCls,
      )}
    >
      <Handle type="target" position={Position.Top} className="!bg-slate-400" />

      <div className="flex items-center gap-2">
        <div className="flex h-7 w-7 items-center justify-center rounded-md bg-slate-100">
          <Terminal className="h-4 w-4 text-slate-600" />
        </div>
        <div className="flex flex-col">
          <span className="text-xs font-medium text-slate-800 leading-tight">
            {d.label || 'Shell Script'}
          </span>
          <span className="text-[10px] text-slate-400">{d.interpreter ?? 'bash'}</span>
        </div>
      </div>

      {/* Success handle */}
      <Handle
        type="source"
        position={Position.Bottom}
        id="success"
        style={{ left: '35%' }}
        className="!bg-green-400"
      />
      {/* Failure handle */}
      <Handle
        type="source"
        position={Position.Bottom}
        id="failure"
        style={{ left: '65%' }}
        className="!bg-red-400"
      />
    </div>
  )
}
