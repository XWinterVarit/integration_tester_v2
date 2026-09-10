import { useState } from 'react'

const STATUS_CLASS = {
  PASSED: 'status-passed',
  FAILED: 'status-failed',
  'Running...': 'status-running',
  'Not Run': 'status-idle',
}

export default function StageTree({ stages, onRunStage, onRunAction, onRunAll, onDiscover }) {
  const [expanded, setExpanded] = useState(() => new Set())

  const toggle = (name) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(name)) {
        next.delete(name)
      } else {
        next.add(name)
      }
      return next
    })
  }

  return (
    <section className="pane stage-pane">
      <div className="pane-header">
        <h2>Test Stages</h2>
        <div className="pane-actions">
          <button className="btn primary" onClick={onRunAll}>
            Run All
          </button>
          <button className="btn" onClick={onDiscover}>
            Refresh Actions
          </button>
        </div>
      </div>

      <div className="stage-list">
        {stages.length === 0 && <p className="empty">No stages registered.</p>}

        {stages.map((stage) => {
          const isOpen = expanded.has(stage.name)
          const actions = stage.actions || []
          return (
            <div className="stage" key={stage.name}>
              <div className="stage-row">
                <button
                  className="disclosure"
                  onClick={() => toggle(stage.name)}
                  aria-label={isOpen ? 'Collapse' : 'Expand'}
                >
                  {isOpen ? '▾' : '▸'}
                </button>
                <span className="stage-name">{stage.name}</span>
                <span className={`status ${STATUS_CLASS[stage.status] || 'status-idle'}`}>
                  {stage.status}
                </span>
                <button className="btn primary small" onClick={() => onRunStage(stage.name)}>
                  Run Stage
                </button>
              </div>

              {isOpen && (
                <ul className="action-list">
                  {actions.length === 0 && (
                    <li className="empty small">
                      No actions recorded. Click “Refresh Actions”.
                    </li>
                  )}
                  {actions.map((action) => (
                    <li className="action-row" key={action.index}>
                      <span className="action-summary">{action.summary}</span>
                      <button
                        className="btn small"
                        onClick={() => onRunAction(stage.name, action.index)}
                      >
                        Run
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )
        })}
      </div>
    </section>
  )
}
