import { useEffect, useRef, useState, useCallback } from 'react'
import { apiClient } from '../api/client'
import type { LogChunk } from '../types'

const MAX_RETRIES = 5
const BASE_BACKOFF_MS = 1000

export function useLogStream(executionId: string | null) {
  const [logs, setLogs] = useState<LogChunk[]>([])
  const [isConnected, setIsConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const wsRef = useRef<WebSocket | null>(null)
  const retriesRef = useRef(0)
  const lastSeqRef = useRef(0)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const fetchExistingLogs = useCallback(async (execId: string, fromSeq: number) => {
    try {
      const res = await apiClient.get<LogChunk[]>(`/executions/${execId}/logs`, {
        params: { fromSeq },
      })
      if (res.data.length > 0) {
        setLogs((prev) => [...prev, ...res.data])
        lastSeqRef.current = res.data[res.data.length - 1].seq
      }
    } catch {
      // ignore - will show whatever the WebSocket delivers
    }
  }, [])

  const connect = useCallback(
    (execId: string) => {
      if (wsRef.current) {
        wsRef.current.close()
      }

      const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
      const token = localStorage.getItem('openote_token') ?? ''
      const url = `${protocol}://${window.location.host}/api/v1/executions/${execId}/logs/stream?fromSeq=${lastSeqRef.current}&token=${token}`

      const ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        setIsConnected(true)
        setError(null)
        retriesRef.current = 0
      }

      ws.onmessage = (event) => {
        try {
          const chunk: LogChunk = JSON.parse(event.data)
          setLogs((prev) => [...prev, chunk])
          lastSeqRef.current = chunk.seq
        } catch {
          // ignore malformed message
        }
      }

      ws.onerror = () => {
        setError('WebSocket connection error')
      }

      ws.onclose = () => {
        setIsConnected(false)
        if (retriesRef.current < MAX_RETRIES) {
          const delay = BASE_BACKOFF_MS * Math.pow(2, retriesRef.current)
          retriesRef.current++
          reconnectTimerRef.current = setTimeout(() => connect(execId), delay)
        } else {
          setError('Connection lost after max retries')
        }
      }
    },
    [],
  )

  useEffect(() => {
    if (!executionId) return

    setLogs([])
    lastSeqRef.current = 0
    retriesRef.current = 0

    // First fetch existing logs, then open WebSocket
    fetchExistingLogs(executionId, 0).then(() => {
      connect(executionId)
    })

    return () => {
      if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current)
      if (wsRef.current) {
        wsRef.current.onclose = null // prevent reconnect on unmount
        wsRef.current.close()
      }
    }
  }, [executionId, connect, fetchExistingLogs])

  return { logs, isConnected, error }
}
