'use client'

import * as React from 'react'
import type { LogEntry, Alert, TrafficPoint } from '@/types'

interface WebSocketMessage {
  type: 'log' | 'alert' | 'traffic' | 'stats'
  data: LogEntry | Alert | TrafficPoint | Record<string, unknown>
}

interface WebSocketContextValue {
  isConnected: boolean
  lastMessage: WebSocketMessage | null
  messages: WebSocketMessage[]
  logs: LogEntry[]
  alerts: Alert[]
  traffic: TrafficPoint[]
  subscribe: (siteId?: string) => void
  unsubscribe: () => void
}

const WebSocketContext = React.createContext<WebSocketContextValue>({
  isConnected: false,
  lastMessage: null,
  messages: [],
  logs: [],
  alerts: [],
  traffic: [],
  subscribe: () => {},
  unsubscribe: () => {},
})

const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/ws'

export function WebSocketProvider({ children }: { children: React.ReactNode }) {
  const [isConnected, setIsConnected] = React.useState(false)
  const [lastMessage, setLastMessage] = React.useState<WebSocketMessage | null>(null)
  const [messages, setMessages] = React.useState<WebSocketMessage[]>([])
  const [logs, setLogs] = React.useState<LogEntry[]>([])
  const [alerts, setAlerts] = React.useState<Alert[]>([])
  const [traffic, setTraffic] = React.useState<TrafficPoint[]>([])
  const wsRef = React.useRef<WebSocket | null>(null)
  const maxItems = 1000

  const connect = React.useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) return

    const token = typeof window !== 'undefined' ? localStorage.getItem('fortresswaf_token') : null
    const ws = new WebSocket(`${WS_URL}?token=${token || ''}`)

    ws.onopen = () => setIsConnected(true)
    ws.onclose = () => {
      setIsConnected(false)
      setTimeout(connect, 3000)
    }
    ws.onerror = () => ws.close()

    ws.onmessage = (event) => {
      try {
        const parsed: WebSocketMessage = JSON.parse(event.data)
        setLastMessage(parsed)
        setMessages((prev) => [...prev.slice(-maxItems), parsed])

        switch (parsed.type) {
          case 'log':
            setLogs((prev) => [...prev.slice(-maxItems), parsed.data as LogEntry])
            break
          case 'alert':
            setAlerts((prev) => [...prev.slice(-50), parsed.data as Alert])
            break
          case 'traffic':
            setTraffic((prev) => [...prev.slice(-500), parsed.data as TrafficPoint])
            break
        }
      } catch (e) {
        console.error('Failed to parse WebSocket message:', e)
      }
    }

    wsRef.current = ws
  }, [])

  const disconnect = React.useCallback(() => {
    wsRef.current?.close()
    wsRef.current = null
    setIsConnected(false)
  }, [])

  const subscribe = React.useCallback((siteId?: string) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: 'subscribe', siteId }))
    }
  }, [])

  const unsubscribe = React.useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: 'unsubscribe' }))
    }
  }, [])

  React.useEffect(() => {
    connect()
    return () => disconnect()
  }, [connect, disconnect])

  return (
    <WebSocketContext.Provider
      value={{ isConnected, lastMessage, messages, logs, alerts, traffic, subscribe, unsubscribe }}
    >
      {children}
    </WebSocketContext.Provider>
  )
}

export function useWebSocket() {
  const context = React.useContext(WebSocketContext)
  if (!context) throw new Error('useWebSocket must be used within a WebSocketProvider')
  return context
}
