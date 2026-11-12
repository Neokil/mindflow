"""Isolated TLS proxy and capturing SMTP servers for container verification."""
import base64
import email.policy
import http.client
import http.server
import json
import socketserver
import ssl
import threading
from email.parser import BytesParser

messages = []
lock = threading.Lock()
context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
context.load_cert_chain('/certs/cert.pem', '/certs/key.pem')

class SMTP(socketserver.StreamRequestHandler):
    def setup(self):
        if self.server.server_address[1] == 1465:
            self.request = context.wrap_socket(self.request, server_side=True)
        super().setup()

    def respond(self, text):
        self.wfile.write((text + '\r\n').encode())
        self.wfile.flush()

    def handle(self):
        self.respond('220 mindflow test SMTP')
        while raw := self.rfile.readline(65536):
            line = raw.decode().strip()
            command = line.split(' ', 1)[0].upper()
            if command in ('EHLO', 'HELO'):
                self.respond('250-fixture\r\n250-STARTTLS\r\n250 AUTH PLAIN')
            elif command == 'STARTTLS':
                self.respond('220 Ready for TLS')
                self.connection = context.wrap_socket(self.connection, server_side=True)
                self.rfile = self.connection.makefile('rb')
                self.wfile = self.connection.makefile('wb')
            elif command == 'AUTH':
                expected = b'\0fixture\0fixture-password'
                valid = base64.b64decode(line.split()[-1]) == expected
                self.respond('235 Authenticated' if valid else '535 Invalid credentials')
            elif command == 'DATA':
                self.respond('354 Finish with a dot')
                data = bytearray()
                while (part := self.rfile.readline(65536)) not in (b'.\r\n', b''):
                    data.extend(part[1:] if part.startswith(b'..') else part)
                message = BytesParser(policy=email.policy.default).parsebytes(bytes(data))
                with lock:
                    messages.append({'to': str(message['To']), 'subject': str(message['Subject']),
                                     'body': message.get_content(), 'tls': isinstance(self.connection, ssl.SSLSocket),
                                     'port': self.server.server_address[1]})
                self.respond('250 Accepted')
            elif command == 'QUIT':
                self.respond('221 Bye')
                return
            else:
                self.respond('250 OK')

class SMTPServer(socketserver.ThreadingTCPServer):
    allow_reuse_address = True
    daemon_threads = True

class Proxy(http.server.BaseHTTPRequestHandler):
    protocol_version = 'HTTP/1.1'

    def log_message(self, *_args):
        pass  # Test tokens and passwords must not enter request logs.

    def forward(self):
        if self.path == '/_messages':
            with lock:
                body = json.dumps(messages).encode()
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.send_header('Content-Length', str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        upstream = http.client.HTTPConnection('app', 8080, timeout=30)
        headers = {key: value for key, value in self.headers.items()
                   if key.lower() not in ('connection', 'transfer-encoding')}
        headers['X-Forwarded-Proto'] = 'https'
        body = self.rfile.read(int(self.headers.get('Content-Length', '0')))
        upstream.request(self.command, self.path, body=body, headers=headers)
        response = upstream.getresponse()
        self.send_response(response.status)
        for key, value in response.getheaders():
            if key.lower() not in ('connection', 'transfer-encoding', 'content-length'):
                self.send_header(key, value)
        if response.getheader('Content-Type', '').startswith('text/event-stream'):
            self.send_header('Connection', 'close')
            self.end_headers()
            try:
                while part := response.read1(4096):
                    self.wfile.write(part)
                    self.wfile.flush()
            except (OSError, TimeoutError):
                pass
        else:
            body = response.read()
            self.send_header('Content-Length', str(len(body)))
            self.send_header('Connection', 'close')
            self.end_headers()
            self.wfile.write(body)
        upstream.close()
        self.close_connection = True

    do_GET = do_POST = do_PUT = do_PATCH = do_DELETE = forward

for port in (1025, 1465):
    server = SMTPServer(('0.0.0.0', port), SMTP)
    threading.Thread(target=server.serve_forever, daemon=True).start()
web = http.server.ThreadingHTTPServer(('0.0.0.0', 8443), Proxy)
web.socket = context.wrap_socket(web.socket, server_side=True)
web.serve_forever()
