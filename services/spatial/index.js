const WebSocket = require('ws')

const server = new WebSocket.Server({ port: 28786 })

server.on('connection', (client) => {
  client.on('message', (message, isBinary) => {
    server.clients.forEach((other) => {
      if (client.readyState !== WebSocket.OPEN || other == client) return
      other.send(message, { binary: isBinary })
    })
  })

  client.on('close', () => {
    console.log(`client disconnected (total: ${server.clients.size})`)
  })

  console.log(`client connected (total: ${server.clients.size})`)
})
