const DEFAULT_SERVER = 'http://127.0.0.1:9101'

function trimSlash(url) {
  return url.replace(/\/+$/, '')
}

export function resolveServerUrl() {
  if (typeof window !== 'undefined') {
    if (window.integrationTester?.serverUrl) {
      return trimSlash(window.integrationTester.serverUrl)
    }
    const fromQuery = new URLSearchParams(window.location.search).get('server')
    if (fromQuery) {
      return trimSlash(fromQuery)
    }
    // Served directly by the Go server (browser fallback): same origin.
    if (window.location.protocol === 'http:' || window.location.protocol === 'https:') {
      return trimSlash(window.location.origin)
    }
  }
  return DEFAULT_SERVER
}

export function resolveToken() {
  if (typeof window !== 'undefined') {
    if (window.integrationTester?.token) {
      return window.integrationTester.token
    }
    const hash = window.location.hash.replace(/^#/, '')
    const fromHash = new URLSearchParams(hash).get('token')
    if (fromHash) {
      return fromHash
    }
    const fromQuery = new URLSearchParams(window.location.search).get('token')
    if (fromQuery) {
      return fromQuery
    }
  }
  return ''
}

export function eventsUrl(base) {
  const token = resolveToken()
  return `${base}/api/events${token ? `?token=${encodeURIComponent(token)}` : ''}`
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
  const headers = {}
  if (options.body) headers['Content-Type'] = 'application/json'
  const token = resolveToken()
  if (token) headers['Authorization'] = `Bearer ${token}`

  const res = await fetch(base + path, {
    method: options.method || 'GET',
    headers,
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
