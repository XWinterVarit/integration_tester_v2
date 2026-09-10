import { useEffect } from 'react'
import { iconFor } from '../logIcons'

export default function LogDetail({ log, onClose }) {
  useEffect(() => {
    const onKeyDown = (event) => {
      if (event.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [onClose])

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(event) => event.stopPropagation()}>
        <header className="modal-header">
          <div className="modal-heading">
            <span className="log-icon">{iconFor(log.type)}</span>
            <span className="modal-title">{log.summary}</span>
          </div>
          <button className="btn small" onClick={onClose}>
            Close
          </button>
        </header>
        <div className="modal-meta">Type: {log.type}</div>
        <pre className="modal-detail">{log.detail || '(No details available)'}</pre>
      </div>
    </div>
  )
}
