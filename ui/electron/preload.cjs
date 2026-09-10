const { contextBridge } = require('electron')

contextBridge.exposeInMainWorld('integrationTester', {
  serverUrl: process.env.IT_SERVER_URL || '',
})
