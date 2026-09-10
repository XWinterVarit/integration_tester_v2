export const LOG_ICONS = {
  Stage: '▦',
  DB: '▤',
  Redis: '◇',
  Request: '↗',
  Mock: '◈',
  App: '▣',
  Expect: '✓',
  Error: '✕',
  Info: 'ⓘ',
}

export function iconFor(type) {
  return LOG_ICONS[type] || '•'
}
