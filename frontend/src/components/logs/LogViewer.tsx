import { useEffect, useRef } from 'react'
import { Terminal as TerminalIcon, Download } from 'lucide-react'
import { useLogStream } from '../../hooks/useLogStream'
import { Button } from '../ui/button'

interface LogViewerProps {
  executionId: string
  autoScroll?: boolean
}

const STREAM_COLORS = {
  stdout: '\x1b[37m',   // white
  stderr: '\x1b[91m',   // bright red
  system: '\x1b[33m',  // yellow
}
const RESET = '\x1b[0m'

export function LogViewer({ executionId, autoScroll = true }: LogViewerProps) {
  const termRef = useRef<HTMLDivElement>(null)
  const terminalRef = useRef<import('@xterm/xterm').Terminal | null>(null)
  const fitAddonRef = useRef<import('@xterm/addon-fit').FitAddon | null>(null)
  const { logs, isConnected, error } = useLogStream(executionId)

  useEffect(() => {
    let terminal: import('@xterm/xterm').Terminal
    let fitAddon: import('@xterm/addon-fit').FitAddon

    async function initTerminal() {
      const { Terminal } = await import('@xterm/xterm')
      const { FitAddon } = await import('@xterm/addon-fit')
      await import('@xterm/xterm/css/xterm.css')

      if (!termRef.current) return

      terminal = new Terminal({
        theme: {
          background: '#0f172a',
          foreground: '#e2e8f0',
          cursor: '#94a3b8',
        },
        fontFamily: '"JetBrains Mono", "Fira Code", monospace',
        fontSize: 13,
        lineHeight: 1.5,
        scrollback: 10000,
        convertEol: true,
      })
      fitAddon = new FitAddon()
      terminal.loadAddon(fitAddon)
      terminal.open(termRef.current)
      fitAddon.fit()

      terminalRef.current = terminal
      fitAddonRef.current = fitAddon

      const ro = new ResizeObserver(() => fitAddon.fit())
      ro.observe(termRef.current)
    }

    initTerminal()

    return () => {
      terminalRef.current?.dispose()
      terminalRef.current = null
    }
  }, [])

  // Write incoming log chunks to terminal
  useEffect(() => {
    const terminal = terminalRef.current
    if (!terminal) return
    for (const chunk of logs) {
      const color = STREAM_COLORS[chunk.stream] ?? STREAM_COLORS.stdout
      terminal.writeln(`${color}${chunk.content}${RESET}`)
    }
    if (autoScroll) {
      terminal.scrollToBottom()
    }
  }, [logs, autoScroll])

  function downloadLogs() {
    const text = logs.map((c) => `[${c.stream}] ${c.content}`).join('\n')
    const blob = new Blob([text], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `execution_${executionId}.log`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="flex flex-col h-full rounded-lg border border-slate-200 overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between bg-slate-800 px-4 py-2">
        <div className="flex items-center gap-2">
          <TerminalIcon className="h-4 w-4 text-slate-400" />
          <span className="text-xs font-medium text-slate-300">実行ログ</span>
          <span
            className={`h-2 w-2 rounded-full ${isConnected ? 'bg-green-400 animate-pulse' : 'bg-slate-500'}`}
          />
        </div>
        <div className="flex items-center gap-2">
          {error && <span className="text-xs text-red-400">{error}</span>}
          <Button variant="ghost" size="icon" onClick={downloadLogs} className="h-7 w-7 text-slate-400 hover:text-white hover:bg-slate-700">
            <Download className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      {/* Terminal area */}
      <div ref={termRef} className="flex-1 bg-slate-900 p-1" />
    </div>
  )
}
