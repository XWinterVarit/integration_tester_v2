const DEFAULT_SERVER = 'http://127.0.0.1:9101'

function trimSlash(url) {
  return url.replace(/\/+$/, '')
}

export function resolveServerUrl() {
  if (typeof window !== 'undefined' && window.integrationTester?.serverUrl) {
    return trimSlash(window.integrationTester.serverUrl)
  }
  const fromQuery = new URLSearchParams(window.location.search).get('server')
  if (fromQuery) {
    return trimSlash(fromQuery)
  }
  return DEFAULT_SERVER
}

export function fetchState(base) {
  return request(base, '/api/state')
}

export function runStage(base, name, runPrerequisites = true) {
  return request(base, '/api/stage/run', {
    method: 'POST',
    body: { name, runPrerequisites },
  })
}

export function runAction(base, stage, index) {
  return request(base, '/api/action/run', {
    method: 'POST',
    body: { stage, index },
  })
}

export function runAll(base) {
  return request(base, '/api/run-all', { method: 'POST', body: {} })
}

export function discover(base) {
  return request(base, '/api/discover', { method: 'POST', body: {} })
}

async function request(base, path, options = {}) {
  const res = await fetch(base + path, {
    method: options.method || 'GET',
    headers: options.body ? { 'Content-Type': 'application/json' } : undefined,
    body: options.body ? JSON.stringify(options.body) : undefined,
  })

  if (!res.ok) {
    let message = `Request failed (${res.status})`
    try {
      const data = await res.json()
      if (data?.error) message = data.error
    } catch {
      // ignore non-JSON error bodies
    }
    throw new Error(message)
  }

  const text = await res.text()
  return text ? JSON.parse(text) : null
}
