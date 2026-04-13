import { Handle, Position, type NodeProps } from '@xyflow/react'
import { Globe } from 'lucide-react'
import { cn } from '../../../lib/utils'
import type { ExecutionStatus } from '../../../types'

export interface HttpRequestNodeData {
  label: string
  nodeDefId?: string
  method?: string
  url?: string
  executionStatus?: ExecutionStatus
  [key: string]: unknown
}

const statusBorder: Partial<Record<ExecutionStatus, string>> = {
  running: 'border-blue-400',
  success: 'border-green-400',
  failed: 'border-red-400',
  aborted: 'border-gray-400',
}

const methodColor: Record<string, string> = {
  GET: 'text-green-600 bg-green-50',
  POST: 'text-blue-600 bg-blue-50',
  PUT: 'text-yellow-600 bg-yellow-50',
  DELETE: 'text-red-600 bg-red-50',
  PATCH: 'text-purple-600 bg-purple-50',
}

export function HttpRequestNode({ data, selected }: NodeProps) {
  const d = data as HttpRequestNodeData
  const statusCls = d.executionStatus ? statusBorder[d.executionStatus] ?? '' : ''
  const method = d.method ?? 'GET'
  const methodCls = methodColor[method] ?? 'text-slate-600 bg-slate-50'

  return (
    <div
      className={cn(
        'rounded-lg border-2 bg-white px-4 py-3 shadow-md transition-all min-w-[160px]',
        selected ? 'border-primary ring-2 ring-primary/30' : 'border-teal-300',
        statusCls,
      )}
    >
      <Handle type="target" position={Position.Top} className="!bg-teal-400" />

      <div className="flex items-center gap-2">
        <div className="flex h-7 w-7 items-center justify-center rounded-md bg-teal-50">
          <Globe className="h-4 w-4 text-teal-600" />
        </div>
        <div className="flex flex-col">
          <span className="text-xs font-medium text-slate-800 leading-tight">
            {d.label || 'HTTP Request'}
          </span>
          <div className="flex items-center gap-1">
            <span className={cn('rounded px-1 text-[9px] font-bold', methodCls)}>{method}</span>
            {d.url && (
              <span className="text-[10px] text-slate-400 max-w-[90px] truncate">{d.url}</span>
            )}
          </div>
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
