import { useNavigate, useParams } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { ArrowLeft, Clock, Save } from 'lucide-react'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Card, CardHeader, CardTitle, CardContent } from '../components/ui/card'
import { useSchedule, useUpsertSchedule, useDeleteSchedule } from '../api/jobs'

const schema = z.object({
  cronExpr: z.string().min(1, 'cron式は必須です'),
  timezone: z.string().default('Asia/Tokyo'),
  isEnabled: z.boolean().default(true),
})
type FormValues = z.infer<typeof schema>

const CRON_EXAMPLES = [
  { label: '毎分', value: '* * * * *' },
  { label: '毎時0分', value: '0 * * * *' },
  { label: '毎日9時', value: '0 9 * * *' },
  { label: '平日9時', value: '0 9 * * 1-5' },
  { label: '毎週月曜9時', value: '0 9 * * 1' },
  { label: '毎月1日9時', value: '0 9 1 * *' },
]

export default function JobSchedulePage() {
  const { pid, jid } = useParams<{ pid: string; jid: string }>()
  const navigate = useNavigate()

  const { data: schedule } = useSchedule(pid ?? '', jid ?? '')
  const upsertSchedule = useUpsertSchedule()
  const deleteSchedule = useDeleteSchedule()

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    values: schedule
      ? { cronExpr: schedule.cronExpr, timezone: schedule.timezone, isEnabled: schedule.isEnabled }
      : undefined,
  })

  const cronExpr = watch('cronExpr', '')

  async function onSubmit(values: FormValues) {
    await upsertSchedule.mutateAsync({ projectId: pid!, jobId: jid!, ...values })
    navigate(-1)
  }

  async function handleDelete() {
    if (!confirm('スケジュールを削除しますか？')) return
    await deleteSchedule.mutateAsync({ projectId: pid!, jobId: jid! })
    navigate(-1)
  }

  return (
    <div className="min-h-screen bg-slate-50 p-6">
      <div className="max-w-xl mx-auto">
        <div className="flex items-center gap-3 mb-6">
          <button onClick={() => navigate(-1)} className="rounded p-1 hover:bg-slate-200">
            <ArrowLeft className="h-4 w-4 text-slate-500" />
          </button>
          <div className="flex items-center gap-2">
            <Clock className="h-5 w-5 text-slate-600" />
            <h1 className="text-lg font-bold text-slate-900">スケジュール設定</h1>
          </div>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>cron スケジュール</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-5">
              {/* Cron expression */}
              <div>
                <label className="mb-1 block text-sm font-medium">cron 式</label>
                <Input
                  {...register('cronExpr')}
                  placeholder="0 9 * * 1-5"
                  className="font-mono"
                />
                {errors.cronExpr && (
                  <p className="mt-1 text-xs text-red-500">{errors.cronExpr.message}</p>
                )}
                {cronExpr && (
                  <p className="mt-1.5 text-xs text-slate-500 font-mono bg-slate-50 px-2 py-1 rounded">
                    {cronExpr}
                  </p>
                )}
              </div>

              {/* Quick examples */}
              <div>
                <p className="mb-2 text-xs font-medium text-slate-500">よく使う例</p>
                <div className="flex flex-wrap gap-1.5">
                  {CRON_EXAMPLES.map((ex) => (
                    <button
                      key={ex.value}
                      type="button"
                      onClick={() => setValue('cronExpr', ex.value)}
                      className="rounded-full border border-slate-200 bg-white px-2.5 py-1 text-xs hover:bg-slate-100 transition-colors"
                    >
                      {ex.label}
                      <span className="ml-1 font-mono text-slate-400">{ex.value}</span>
                    </button>
                  ))}
                </div>
              </div>

              {/* Timezone */}
              <div>
                <label className="mb-1 block text-sm font-medium">タイムゾーン</label>
                <select
                  {...register('timezone')}
                  className="w-full rounded-md border border-input px-3 py-2 text-sm"
                >
                  <option value="Asia/Tokyo">Asia/Tokyo (JST)</option>
                  <option value="UTC">UTC</option>
                  <option value="America/New_York">America/New_York (EST)</option>
                  <option value="America/Los_Angeles">America/Los_Angeles (PST)</option>
                  <option value="Europe/London">Europe/London (GMT)</option>
                  <option value="Europe/Berlin">Europe/Berlin (CET)</option>
                </select>
              </div>

              {/* Enabled */}
              <label className="flex items-center gap-2 cursor-pointer">
                <input type="checkbox" {...register('isEnabled')} className="h-4 w-4" />
                <span className="text-sm font-medium">スケジュールを有効にする</span>
              </label>

              {/* Next run (read-only) */}
              {schedule?.nextRunAt && (
                <p className="text-xs text-slate-500">
                  次回実行予定:{' '}
                  <span className="font-mono">
                    {new Date(schedule.nextRunAt).toLocaleString('ja-JP')}
                  </span>
                </p>
              )}

              <div className="flex gap-2 pt-2">
                <Button type="submit" disabled={isSubmitting}>
                  <Save className="h-4 w-4 mr-1" />
                  {isSubmitting ? '保存中...' : '保存'}
                </Button>
                {schedule && (
                  <Button
                    type="button"
                    variant="destructive"
                    onClick={handleDelete}
                    disabled={deleteSchedule.isPending}
                  >
                    削除
                  </Button>
                )}
                <Button type="button" variant="outline" onClick={() => navigate(-1)}>
                  キャンセル
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
