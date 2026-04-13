import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, FolderOpen, LogOut } from 'lucide-react'
import { Button } from '../components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '../components/ui/card'
import { Dialog, DialogHeader, DialogTitle, DialogContent, DialogFooter } from '../components/ui/dialog'
import { Input } from '../components/ui/input'
import { JobList } from '../components/jobs/JobList'
import { useProjects, useCreateProject } from '../api/projects'
import { useAuthStore } from '../store/auth'

export default function DashboardPage() {
  const navigate = useNavigate()
  const logout = useAuthStore((s) => s.logout)
  const user = useAuthStore((s) => s.user)
  const { data: projects = [], isLoading } = useProjects()
  const createProject = useCreateProject()

  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(null)
  const [showNewProject, setShowNewProject] = useState(false)
  const [newProjectName, setNewProjectName] = useState('')
  const [newProjectDesc, setNewProjectDesc] = useState('')

  const selectedProject = projects.find((p) => p.id === selectedProjectId)

  async function handleCreateProject() {
    if (!newProjectName.trim()) return
    const project = await createProject.mutateAsync({
      name: newProjectName.trim(),
      description: newProjectDesc.trim(),
    })
    setSelectedProjectId(project.id)
    setShowNewProject(false)
    setNewProjectName('')
    setNewProjectDesc('')
  }

  return (
    <div className="flex h-screen bg-slate-50">
      {/* Sidebar */}
      <aside className="w-64 flex-shrink-0 flex flex-col border-r bg-white">
        <div className="flex items-center justify-between border-b px-4 py-4">
          <span className="font-bold text-lg text-slate-900">Openote</span>
        </div>

        <div className="flex-1 overflow-y-auto p-3">
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-semibold text-slate-500 uppercase tracking-wide">
              プロジェクト
            </span>
            <button
              onClick={() => setShowNewProject(true)}
              className="rounded p-0.5 hover:bg-slate-100"
              title="プロジェクトを作成"
            >
              <Plus className="h-4 w-4 text-slate-500" />
            </button>
          </div>

          {isLoading && (
            <p className="text-xs text-slate-400 px-2">読み込み中...</p>
          )}

          {projects.map((p) => (
            <button
              key={p.id}
              onClick={() => setSelectedProjectId(p.id)}
              className={`flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-left transition-colors ${
                selectedProjectId === p.id
                  ? 'bg-primary text-primary-foreground'
                  : 'text-slate-700 hover:bg-slate-100'
              }`}
            >
              <FolderOpen className="h-4 w-4 flex-shrink-0" />
              <span className="truncate">{p.name}</span>
            </button>
          ))}

          {projects.length === 0 && !isLoading && (
            <p className="text-xs text-slate-400 px-2 mt-2">
              プロジェクトがありません
            </p>
          )}
        </div>

        {/* Footer */}
        <div className="border-t p-3">
          <div className="flex items-center justify-between">
            <span className="text-xs text-slate-500">{user?.username}</span>
            <button
              onClick={logout}
              className="rounded p-1 hover:bg-slate-100"
              title="ログアウト"
            >
              <LogOut className="h-4 w-4 text-slate-400" />
            </button>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto p-6">
        {selectedProject ? (
          <div>
            <div className="mb-6 flex items-center justify-between">
              <div>
                <h1 className="text-xl font-bold text-slate-900">{selectedProject.name}</h1>
                {selectedProject.description && (
                  <p className="text-sm text-slate-500 mt-0.5">{selectedProject.description}</p>
                )}
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => navigate(`/projects/${selectedProject.id}/executions`)}
                >
                  実行履歴
                </Button>
                <Button
                  size="sm"
                  onClick={() => navigate(`/projects/${selectedProject.id}/jobs/new`)}
                >
                  <Plus className="h-4 w-4 mr-1" />
                  新規ジョブ
                </Button>
              </div>
            </div>

            <JobList projectId={selectedProject.id} />
          </div>
        ) : (
          <div className="flex h-full items-center justify-center">
            <Card className="max-w-sm w-full">
              <CardHeader>
                <CardTitle>プロジェクトを選択</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-slate-500 mb-4">
                  左のサイドバーからプロジェクトを選択するか、新規プロジェクトを作成してください。
                </p>
                <Button onClick={() => setShowNewProject(true)}>
                  <Plus className="h-4 w-4 mr-1" />
                  プロジェクトを作成
                </Button>
              </CardContent>
            </Card>
          </div>
        )}
      </main>

      {/* New project dialog */}
      <Dialog open={showNewProject} onClose={() => setShowNewProject(false)}>
        <DialogHeader>
          <DialogTitle>新規プロジェクト</DialogTitle>
        </DialogHeader>
        <DialogContent>
          <div className="flex flex-col gap-3">
            <div>
              <label className="mb-1 block text-sm font-medium">プロジェクト名</label>
              <Input
                value={newProjectName}
                onChange={(e) => setNewProjectName(e.target.value)}
                placeholder="my-project"
                autoFocus
              />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium">説明（任意）</label>
              <Input
                value={newProjectDesc}
                onChange={(e) => setNewProjectDesc(e.target.value)}
                placeholder="プロジェクトの説明"
              />
            </div>
          </div>
        </DialogContent>
        <DialogFooter>
          <Button variant="outline" onClick={() => setShowNewProject(false)}>
            キャンセル
          </Button>
          <Button
            onClick={handleCreateProject}
            disabled={!newProjectName.trim() || createProject.isPending}
          >
            {createProject.isPending ? '作成中...' : '作成'}
          </Button>
        </DialogFooter>
      </Dialog>
    </div>
  )
}
