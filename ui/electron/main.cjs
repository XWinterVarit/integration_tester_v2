const { app, BrowserWindow, shell } = require('electron')
const path = require('node:path')

const serverUrl = process.env.IT_SERVER_URL || ''
const serverToken = process.env.IT_TOKEN || ''
const devServerUrl = process.env.VITE_DEV_SERVER_URL || ''
const indexHtml = process.env.IT_UI_DIST || path.join(__dirname, '..', 'dist', 'index.html')

function createWindow() {
  const win = new BrowserWindow({
    width: 1200,
    height: 800,
    backgroundColor: '#0f1420',
    title: 'Integration Tester',
    webPreferences: {
      preload: path.join(__dirname, 'preload.cjs'),
      contextIsolation: true,
      nodeIntegration: false,
    },
  })

  if (devServerUrl) {
    const url = new URL(devServerUrl)
    url.searchParams.set('server', serverUrl)
    url.searchParams.set('token', serverToken)
    win.loadURL(url.toString())
    win.webContents.openDevTools({ mode: 'detach' })
  } else {
    win.loadFile(indexHtml, { query: { server: serverUrl, token: serverToken } })
  }

  // Open external links in the system browser instead of a new Electron window.
  win.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url)
    return { action: 'deny' }
  })

  return win
}

app.whenReady().then(() => {
  createWindow()

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow()
    }
  })
})

app.on('window-all-closed', () => {
  app.quit()
})
