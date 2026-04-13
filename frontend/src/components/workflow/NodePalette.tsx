import { Terminal, TerminalSquare, Globe } from 'lucide-react'

interface PaletteItem {
  type: string
  label: string
  description: string
  icon: React.ReactNode
  color: string
}

const PALETTE_ITEMS: PaletteItem[] = [
  {
    type: 'shellScript',
    label: 'Shell Script',
    description: 'スクリプトファイルを実行',
    icon: <Terminal className="h-4 w-4" />,
    color: 'bg-slate-100 text-slate-700 border-slate-300',
  },
  {
    type: 'shellCommand',
    label: 'Shell Command',
    description: 'コマンドを直接実行',
    icon: <TerminalSquare className="h-4 w-4" />,
    color: 'bg-indigo-50 text-indigo-700 border-indigo-300',
  },
  {
    type: 'httpRequest',
    label: 'HTTP Request',
    description: 'HTTPエンドポイントを呼び出す',
    icon: <Globe className="h-4 w-4" />,
    color: 'bg-teal-50 text-teal-700 border-teal-300',
  },
]

export function NodePalette() {
  function onDragStart(event: React.DragEvent, nodeType: string) {
    event.dataTransfer.setData('application/reactflow', nodeType)
    event.dataTransfer.effectAllowed = 'move'
  }

  return (
    <div className="flex flex-col gap-2 p-3 w-52 bg-white border-r border-slate-200 overflow-y-auto">
      <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
        ノードパレット
      </p>
      {PALETTE_ITEMS.map((item) => (
        <div
          key={item.type}
          draggable
          onDragStart={(e) => onDragStart(e, item.type)}
          className={`flex cursor-grab items-start gap-2 rounded-lg border p-2.5 select-none hover:shadow-sm active:cursor-grabbing ${item.color}`}
        >
          <div className="mt-0.5 flex-shrink-0">{item.icon}</div>
          <div>
            <div className="text-xs font-medium leading-tight">{item.label}</div>
            <div className="text-[10px] opacity-70 mt-0.5">{item.description}</div>
          </div>
        </div>
      ))}

      <div className="mt-4 rounded-lg bg-slate-50 p-2.5 text-[10px] text-slate-500 leading-relaxed">
        ノードをキャンバスにドラッグ&ドロップして配置してください。
        ノードの下部ハンドルを接続して実行順序を定義します。
      </div>
    </div>
  )
}
