# wsproxy

websocket tcp/udp proxy (based on websockify-c)

```Usage: wsproxy [options] [source_addr:]source_port

  --verbose|-v         verbose messages and per frame traffic
  --daemon|-d          become a daemon (background process)
  --whitelist-hosts|-W LIST  new-line separated target host whitelist file
  --whitelist-ports|-P LIST  new-line separated target port whitelist file
  --pid|-p             desired path of pid file. Default: '/var/run/websockify.pid'

```

Patch for emscripten:

https://github.com/FWGS/emscripten/commit/efcb8ecd0807c5590637812a29b4d1c7cd582719