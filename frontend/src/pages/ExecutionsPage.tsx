import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { ExecutionHistory } from '../components/executions/ExecutionHistory'

export default function ExecutionsPage() {
  const { pid } = useParams<{ pid: string }>()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const initialExecId = searchParams.get('execId') ?? undefined

  return (
    <div className="flex h-screen flex-col bg-slate-50">
      <header className="flex items-center gap-3 border-b bg-white px-4 py-3 flex-shrink-0">
        <button onClick={() => navigate('/dashboard')} className="rounded p-1 hover:bg-slate-100">
          <ArrowLeft className="h-4 w-4 text-slate-500" />
        </button>
        <h1 className="text-base font-bold text-slate-900">実行履歴</h1>
      </header>

      <main className="flex-1 overflow-hidden p-4">
        {pid && (
          <ExecutionHistory projectId={pid} initialExecutionId={initialExecId} />
        )}
      </main>
    </div>
  )
}
