import { Handle, Position, type NodeProps } from '@xyflow/react'
import { TerminalSquare } from 'lucide-react'
import { cn } from '../../../lib/utils'
import type { ExecutionStatus } from '../../../types'

export interface ShellCommandNodeData {
  label: string
  nodeDefId?: string
  command?: string
  executionStatus?: ExecutionStatus
  [key: string]: unknown
}

const statusBorder: Partial<Record<ExecutionStatus, string>> = {
  running: 'border-blue-400',
  success: 'border-green-400',
  failed: 'border-red-400',
  aborted: 'border-gray-400',
}

export function ShellCommandNode({ data, selected }: NodeProps) {
  const d = data as ShellCommandNodeData
  const statusCls = d.executionStatus ? statusBorder[d.executionStatus] ?? '' : ''

  return (
    <div
      className={cn(
        'rounded-lg border-2 bg-white px-4 py-3 shadow-md transition-all min-w-[160px]',
        selected ? 'border-primary ring-2 ring-primary/30' : 'border-indigo-300',
        statusCls,
      )}
    >
      <Handle type="target" position={Position.Top} className="!bg-indigo-400" />

      <div className="flex items-center gap-2">
        <div className="flex h-7 w-7 items-center justify-center rounded-md bg-indigo-50">
          <TerminalSquare className="h-4 w-4 text-indigo-600" />
        </div>
        <div className="flex flex-col">
          <span className="text-xs font-medium text-slate-800 leading-tight">
            {d.label || 'Shell Command'}
          </span>
          {d.command && (
            <span className="text-[10px] text-slate-400 max-w-[120px] truncate font-mono">
              {d.command}
            </span>
          )}
        </div>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        id="success"
        style={{ left: '35%' }}
        className="!bg-green-400"
      />
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
