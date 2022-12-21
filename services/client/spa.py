#!/usr/bin/env python

# From https://gist.githubusercontent.com/iktakahiro/2c48962561ea724f1e9d
# Inspired by https://gist.github.com/jtangelder/e445e9a7f5e31c220be6
# Python3 http.server for Single Page Application

import urllib.parse
import http.server
import socketserver
from pathlib import Path

HOST = ('0.0.0.0', 1235)

class Handler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        url_parts = urllib.parse.urlparse(self.path)
        request_file_path = Path(url_parts.path.strip("/"))

        if not request_file_path.is_file():
            self.path = 'index.html'

        return http.server.SimpleHTTPRequestHandler.do_GET(self)


httpd = socketserver.TCPServer(HOST, Handler)

try:
    httpd.serve_forever()
except Exception:
    httpd.socket.close()
    httpd.shutdown()
