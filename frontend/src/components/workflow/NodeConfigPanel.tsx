import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { X } from 'lucide-react'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { useWorkflowStore } from '../../store/workflow'
import type { Node } from '@xyflow/react'

// ── Schemas ──────────────────────────────────────────────────────────────────

const shellScriptSchema = z.object({
  label: z.string().min(1, '名前は必須です'),
  interpreter: z.string().default('bash'),
  scriptContent: z.string().min(1, 'スクリプト内容は必須です'),
  timeoutSecs: z.coerce.number().int().positive().default(3600),
  workingDir: z.string().optional(),
})

const shellCommandSchema = z.object({
  label: z.string().min(1, '名前は必須です'),
  command: z.string().min(1, 'コマンドは必須です'),
  timeoutSecs: z.coerce.number().int().positive().default(3600),
  workingDir: z.string().optional(),
})

const httpRequestSchema = z.object({
  label: z.string().min(1, '名前は必須です'),
  method: z.enum(['GET', 'POST', 'PUT', 'DELETE', 'PATCH']).default('GET'),
  url: z.string().url('有効なURLを入力してください'),
  timeoutSecs: z.coerce.number().int().positive().default(30),
})

// ── Component ─────────────────────────────────────────────────────────────────

interface NodeConfigPanelProps {
  node: Node
  onClose: () => void
}

export function NodeConfigPanel({ node, onClose }: NodeConfigPanelProps) {
  const updateNode = useWorkflowStore((s) => s.updateNodeData)

  const nodeType = node.type as string

  const schema =
    nodeType === 'shellScript'
      ? shellScriptSchema
      : nodeType === 'shellCommand'
        ? shellCommandSchema
        : httpRequestSchema

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm({ resolver: zodResolver(schema) })

  useEffect(() => {
    reset(node.data as Record<string, unknown>)
  }, [node.id, reset])

  function onSubmit(values: Record<string, unknown>) {
    updateNode(node.id, values)
    onClose()
  }

  return (
    <div className="flex w-72 flex-col bg-white border-l border-slate-200 overflow-y-auto">
      <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3">
        <span className="text-sm font-semibold text-slate-800">ノード設定</span>
        <button onClick={onClose} className="rounded p-1 hover:bg-slate-100">
          <X className="h-4 w-4 text-slate-500" />
        </button>
      </div>

      <form onSubmit={handleSubmit(onSubmit as Parameters<typeof handleSubmit>[0])} className="flex flex-col gap-4 p-4">
        {/* Common: label */}
        <div>
          <label className="label-xs">ノード名</label>
          <Input {...register('label')} placeholder="ノード名" />
          {errors.label && <p className="err">{String(errors.label.message)}</p>}
        </div>

        {/* Shell Script */}
        {nodeType === 'shellScript' && (
          <>
            <div>
              <label className="label-xs">インタープリタ</label>
              <select {...register('interpreter')} className="input-select">
                <option value="bash">bash</option>
                <option value="sh">sh</option>
                <option value="python3">python3</option>
                <option value="node">node</option>
              </select>
            </div>
            <div>
              <label className="label-xs">スクリプト内容</label>
              <textarea
                {...register('scriptContent')}
                rows={8}
                className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-xs font-mono shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                placeholder="#!/bin/bash&#10;echo 'Hello, World!'"
              />
              {errors.scriptContent && <p className="err">{String(errors.scriptContent.message)}</p>}
            </div>
          </>
        )}

        {/* Shell Command */}
        {nodeType === 'shellCommand' && (
          <div>
            <label className="label-xs">コマンド</label>
            <Input {...register('command')} placeholder="echo hello" className="font-mono text-xs" />
            {errors.command && <p className="err">{String(errors.command.message)}</p>}
          </div>
        )}

        {/* HTTP Request */}
        {nodeType === 'httpRequest' && (
          <>
            <div>
              <label className="label-xs">HTTPメソッド</label>
              <select {...register('method')} className="input-select">
                {['GET', 'POST', 'PUT', 'DELETE', 'PATCH'].map((m) => (
                  <option key={m}>{m}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="label-xs">URL</label>
              <Input {...register('url')} placeholder="https://example.com/api" />
              {errors.url && <p className="err">{String(errors.url.message)}</p>}
            </div>
          </>
        )}

        {/* Common: timeout */}
        <div>
          <label className="label-xs">タイムアウト（秒）</label>
          <Input type="number" {...register('timeoutSecs')} defaultValue={3600} />
        </div>

        {/* Common: working dir (shell only) */}
        {(nodeType === 'shellScript' || nodeType === 'shellCommand') && (
          <div>
            <label className="label-xs">作業ディレクトリ（任意）</label>
            <Input {...register('workingDir')} placeholder="/tmp" />
          </div>
        )}

        <Button type="submit" size="sm" className="mt-2">
          保存
        </Button>
      </form>

      <style>{`
        .label-xs { display:block; font-size:0.7rem; font-weight:600; color:#64748b; text-transform:uppercase; letter-spacing:.04em; margin-bottom:4px; }
        .err { font-size:0.7rem; color:#ef4444; margin-top:2px; }
        .input-select { width:100%; border-radius:6px; border:1px solid #e2e8f0; padding:6px 8px; font-size:0.875rem; background:white; }
      `}</style>
    </div>
  )
}
