import { useCallback, useEffect, useRef, useState } from 'react'
import {
  discover,
  eventsUrl,
  fetchState,
  resolveServerUrl,
  runAction,
  runAll,
  runStage,
} from './api'
import StageTree from './components/StageTree'
import LogPanel from './components/LogPanel'
import LogDetail from './components/LogDetail'

export default function App() {
  const [serverUrl] = useState(resolveServerUrl)
  const [stages, setStages] = useState([])
  const [logs, setLogs] = useState([])
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState('')
  const [selectedLog, setSelectedLog] = useState(null)
  const refreshTimer = useRef(null)

  const refresh = useCallback(async () => {
    try {
      const state = await fetchState(serverUrl)
      setStages(state?.stages || [])
      setLogs(state?.logs || [])
    } catch (err) {
      setError(err.message)
    }
  }, [serverUrl])

  useEffect(() => {
    refresh()
  }, [refresh])

  useEffect(() => {
    const source = new EventSource(eventsUrl(serverUrl))

    source.onopen = () => {
      setConnected(true)
      setError('')
    }
    source.onerror = () => setConnected(false)

    source.onmessage = (event) => {
      let payload
      try {
        payload = JSON.parse(event.data)
      } catch {
        return
      }

      switch (payload.type) {
        case 'state':
          setStages(payload.state?.stages || [])
          setLogs(payload.state?.logs || [])
          break
        case 'log':
          if (payload.log) setLogs((prev) => [...prev, payload.log])
          break
        case 'stage':
          setStages((prev) =>
            prev.map((stage) =>
              stage.name === payload.stage ? { ...stage, status: payload.status } : stage,
            ),
          )
          break
        case 'refresh':
          if (refreshTimer.current) clearTimeout(refreshTimer.current)
          refreshTimer.current = setTimeout(refresh, 250)
          break
        case 'error':
          setError(payload.message || 'An error occurred')
          break
        default:
          break
      }
    }

    return () => {
      source.close()
      if (refreshTimer.current) clearTimeout(refreshTimer.current)
    }
  }, [serverUrl, refresh])

  const handleRunStage = useCallback(
    async (name) => {
      try {
        await runStage(serverUrl, name, true)
      } catch (err) {
        setError(err.message)
      }
    },
    [serverUrl],
  )

  const handleRunAction = useCallback(
    async (stage, index) => {
      try {
        await runAction(serverUrl, stage, index)
      } catch (err) {
        setError(err.message)
      }
    },
    [serverUrl],
  )

  const handleRunAll = useCallback(async () => {
    try {
      await runAll(serverUrl)
    } catch (err) {
      setError(err.message)
    }
  }, [serverUrl])

  const handleDiscover = useCallback(async () => {
    try {
      await discover(serverUrl)
    } catch (err) {
      setError(err.message)
    }
  }, [serverUrl])

  return (
    <div className="app">
      <header className="app-header">
        <div className="brand">
          <span className="brand-mark">IT</span>
          <h1>Integration Tester</h1>
        </div>
        <div className="connection">
          <span className={`dot ${connected ? 'online' : 'offline'}`} />
          <span>{connected ? 'Connected' : 'Disconnected'}</span>
          <code>{serverUrl}</code>
        </div>
      </header>

      {error && (
        <div className="error-banner">
          <span>{error}</span>
          <button className="btn small" onClick={() => setError('')}>
            Dismiss
          </button>
        </div>
      )}

      <main className="app-body">
        <StageTree
          stages={stages}
          onRunStage={handleRunStage}
          onRunAction={handleRunAction}
          onRunAll={handleRunAll}
          onDiscover={handleDiscover}
        />
        <LogPanel logs={logs} onSelect={setSelectedLog} />
      </main>

      {selectedLog && <LogDetail log={selectedLog} onClose={() => setSelectedLog(null)} />}
    </div>
  )
}
