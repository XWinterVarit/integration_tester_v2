import { useEffect, useMemo, useRef, useState } from 'react'
import { iconFor } from '../logIcons'

function buildGroups(logs) {
  const groups = []
  let current = null
  let root = null

  logs.forEach((log, index) => {
    if (log.type === 'Stage') {
      current = { id: `stage-${index}`, stage: log, children: [] }
      groups.push(current)
    } else if (current) {
      current.children.push({ log, index })
    } else {
      if (!root) {
        root = { id: 'root', stage: null, children: [] }
        groups.push(root)
      }
      root.children.push({ log, index })
    }
  })

  return groups
}

export default function LogPanel({ logs, onSelect }) {
  const [collapsed, setCollapsed] = useState(() => new Set())
  const endRef = useRef(null)

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' })
  }, [logs.length])

  const groups = useMemo(() => buildGroups(logs), [logs])

  const toggle = (id) => {
    setCollapsed((prev) => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  const renderRow = (log, key, indented) => (
    <button
      key={key}
      className={`log-row${indented ? ' indented' : ''}${log.type === 'Error' ? ' is-error' : ''}`}
      onClick={() => onSelect(log)}
      title="Click to view details"
    >
      <span className="log-icon">{iconFor(log.type)}</span>
      <span className="log-summary">{log.summary}</span>
      <span className="log-type">{log.type}</span>
    </button>
  )

  return (
    <section className="pane log-pane">
      <div className="pane-header">
        <h2>Operation Logs</h2>
        <span className="pane-count">{logs.length} entries</span>
      </div>

      <div className="log-list">
        {logs.length === 0 && <p className="empty">No logs yet.</p>}

        {groups.map((group) => {
          if (!group.stage) {
            return group.children.map((child, i) => renderRow(child.log, `root-${i}`, false))
          }

          const isCollapsed = collapsed.has(group.id)
          return (
            <div className="log-group" key={group.id}>
              <button
                className={`log-row stage-log${group.stage.type === 'Error' ? ' is-error' : ''}`}
                onClick={() => (group.children.length ? toggle(group.id) : onSelect(group.stage))}
              >
                <span className="log-icon">{isCollapsed ? '▸' : '▾'}</span>
                <span className="log-icon">{iconFor(group.stage.type)}</span>
                <span className="log-summary">{group.stage.summary}</span>
                <span className="log-type">{group.stage.type}</span>
              </button>
              {!isCollapsed &&
                group.children.map((child, i) => renderRow(child.log, `${group.id}-${i}`, true))}
            </div>
          )
        })}

        <div ref={endRef} />
      </div>
    </section>
  )
}
