#!/usr/bin/env python3
"""Verify the built image with disposable volumes, HTTPS, and TLS SMTP.

No production container, dataset, tunnel, account, or mail provider is used.
Requires Docker, Python 3, and openssl. Run from the repository root.
"""
import argparse
import base64
import http.cookiejar
import json
import os
import re
import ssl
import struct
import subprocess
import tempfile
import time
import urllib.error
import urllib.request
import uuid
import zlib
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument('--image', default='mindflow:verification')
args = parser.parse_args()
name = 'mindflow-verify-' + uuid.uuid4().hex[:10]
network, app, fixture = name + '-net', name + '-app', name + '-fixture'
volume, backup = name + '-data', name + '-backup'

def docker(*arguments):
    return subprocess.check_output(['docker', *arguments], text=True, stderr=subprocess.PIPE).strip()

def wait_for(action, timeout=30):
    end = time.monotonic() + timeout
    last = None
    while time.monotonic() < end:
        try:
            value = action()
            if value:
                return value
        except (urllib.error.URLError, OSError, AssertionError, json.JSONDecodeError) as error:
            last = error
        time.sleep(.25)
    raise AssertionError(f'Timed out waiting for verification condition: {last}')

def png(width, height):
    def chunk(kind, body):
        return struct.pack('!I', len(body)) + kind + body + struct.pack('!I', zlib.crc32(kind + body))
    pixels = (b'\0' + b'\xa5\x90\xd5\xff' * width) * height
    return b'\x89PNG\r\n\x1a\n' + chunk(b'IHDR', struct.pack('!2I5B', width, height, 8, 6, 0, 0, 0)) + chunk(b'IDAT', zlib.compress(pixels)) + chunk(b'IEND', b'')

try:
    docker('network', 'create', network)
    docker('volume', 'create', volume)
    docker('volume', 'create', backup)
    Path('.cache').mkdir(exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='container-', dir='.cache') as temporary:
        certs = Path(temporary).resolve()
        subprocess.run(['openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes', '-days', '1',
                        '-subj', '/CN=mindflow-verification', '-addext', 'subjectAltName=DNS:mail,DNS:localhost,IP:127.0.0.1',
                        '-keyout', str(certs / 'key.pem'), '-out', str(certs / 'cert.pem')], check=True, capture_output=True)
        os.chmod(certs / 'cert.pem', 0o644)
        docker('build', '-q', '-t', 'mindflow:test-fixture', 'tests/container')
        docker('run', '-d', '--name', fixture, '--network', network, '--network-alias', 'mail',
               '-p', '127.0.0.1::8443', '-v', f'{certs}:/certs:ro', 'mindflow:test-fixture')
        port = docker('port', fixture, '8443/tcp').split(':')[-1]
        public = f'https://localhost:{port}'
        tls = ssl.create_default_context(cafile=str(certs / 'cert.pem'))
        jar = http.cookiejar.CookieJar()
        opener = urllib.request.build_opener(urllib.request.HTTPSHandler(context=tls), urllib.request.HTTPCookieProcessor(jar))
        csrf = ''

        def start(data):
            docker('run', '-d', '--name', app, '--network', network, '--network-alias', 'app',
                   '--read-only', '--cap-drop', 'ALL', '--security-opt', 'no-new-privileges:true',
                   '--tmpfs', '/tmp:size=64m,mode=1777', '-v', f'{data}:/data',
                   '-v', f'{certs / "cert.pem"}:/etc/test-ca.pem:ro', '-e', 'SSL_CERT_FILE=/etc/test-ca.pem', args.image)

        def request(path, method='GET', data=None, expected=200, headers=None, client=None):
            raw = json.dumps(data).encode() if data is not None and not isinstance(data, bytes) else data
            req_headers = {'Origin': public, **({'Content-Type': 'application/json'} if data is not None else {})}
            if csrf:
                req_headers['X-CSRF-Token'] = csrf
            req_headers.update(headers or {})
            req = urllib.request.Request(public + path, data=raw, headers=req_headers, method=method)
            try:
                response = (client or opener).open(req, timeout=15)
            except urllib.error.HTTPError as error:
                response = error
            body = response.read()
            assert response.status == expected, (path, response.status, body)
            parsed = json.loads(body) if response.headers.get('Content-Type', '').startswith('application/json') else body
            return parsed, response.headers

        def call(path, method='GET', data=None, **options):
            return request('/api/v1' + path, method, data, **options)[0]

        def mail_to(email, subject):
            return wait_for(lambda: next((m for m in reversed(request('/_messages')[0]) if m['to'] == email and subject in m['subject']), None))

        def login():
            global csrf
            result, headers = request('/api/v1/auth/login', 'POST', {'email': 'owner@example.invalid', 'password': 'container test password'})
            cookie = headers.get('Set-Cookie', '')
            assert all(flag in cookie for flag in ('Secure', 'HttpOnly', 'SameSite=Lax'))
            csrf = result['csrf']

        start(volume)
        wait_for(lambda: request('/health/ready')[0] == {'ok': True})
        startup = subprocess.run(['docker', 'logs', app], capture_output=True, text=True, check=True)
        credential = re.search(r'credential \(operator only\): ([a-f0-9]+)', startup.stdout + startup.stderr)[1]
        settings = {'publicURL': public, 'smtp': {'host': 'mail', 'port': 1465, 'mode': 'tls', 'username': 'fixture', 'password': 'fixture-password', 'from': 'mindflow@example.invalid', 'name': 'mindflow'}}
        call('/setup/complete', 'POST', {'credential': credential, 'email': 'owner@example.invalid', 'password': 'container test password', 'settings': settings})
        login()
        call('/admin/settings/test-email', 'POST', {})
        assert mail_to('owner@example.invalid', 'test email')['tls']
        call('/admin/users', 'POST', {'email': 'friend@example.invalid', 'role': 'admin'})
        invite = mail_to('friend@example.invalid', 'invited')
        token = re.search(r'#accept-invite/([a-f0-9]+)', invite['body'])[1]
        call('/auth/accept-invite', 'POST', {'token': token, 'password': 'friend test password'})
        friend = urllib.request.build_opener(urllib.request.HTTPSHandler(context=tls), urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
        call('/auth/login', 'POST', {'email': 'friend@example.invalid', 'password': 'friend test password'}, client=friend)
        created = call('/workspaces', 'POST', {'name': 'Container verification'})
        workspace, page = created['id'], created['pageId']
        call(f'/workspaces/{workspace}/pages', client=friend, expected=404)
        original = png(1024, 512)
        boundary = uuid.uuid4().hex
        multipart = (f'--{boundary}\r\nContent-Disposition: form-data; name="file"; filename="test.png"\r\nContent-Type: image/png\r\n\r\n'.encode() + original + f'\r\n--{boundary}--\r\n'.encode())
        asset = call('/assets', 'POST', multipart, headers={'Content-Type': f'multipart/form-data; boundary={boundary}'})
        assert request('/api/v1/assets/' + asset['id'])[0] == original
        call('/assets/' + asset['id'], client=friend, expected=404)
        lease = call(f'/pages/{page}/lease', 'POST', {'clientId': 'container-editor', 'action': 'acquire'})
        document = {'schemaVersion': 1, 'settings': {'background': 'dots', 'gridSize': 20, 'snap': False}, 'elements': [{'id': 'image', 'type': 'image', 'x': 0, 'y': 0, 'width': 100, 'height': 50, 'rotation': 0, 'assetId': asset['id']}]}
        mutation = {'baseRevision': 0, 'epoch': lease['epoch'], 'clientId': 'container-editor', 'mutationId': 'container-save', 'document': document}
        event_request = urllib.request.Request(public + '/api/v1/events', headers={'Origin': public})
        with opener.open(event_request, timeout=10) as stream:
            assert stream.headers['Cache-Control'] == 'no-store'
            assert stream.headers['Content-Type'].startswith('text/event-stream')
            assert b'connected' in stream.readline()
            stream.readline()  # Empty line terminating the initial SSE event.
            call(f'/pages/{page}/document', 'PUT', mutation)
            while True:
                line = stream.readline()
                assert line, 'SSE connection ended before a save notification'
                if line.startswith(b'data:') and page.encode() in line:
                    break
        call(f'/pages/{page}/document', 'PUT', mutation)
        assert call(f'/pages/{page}/document')['revision'] == 1
        wait_for(lambda: call('/assets/' + asset['id'] + '/meta')['variants'])
        readonly = call(f'/pages/{page}/lease', 'POST', {'clientId': 'second-editor', 'action': 'acquire'})
        assert not readonly['editable']
        call(f'/pages/{page}/lease', 'POST', {'clientId': 'second-editor', 'action': 'takeover'})
        call(f'/pages/{page}/document', 'PUT', {**mutation, 'baseRevision': 1, 'mutationId': 'stale-editor-save'}, expected=409)
        settings['smtp'].update({'port': 1025, 'mode': 'starttls'})
        call('/admin/settings', 'PUT', settings)
        call('/admin/settings/test-email', 'POST', {})
        assert mail_to('owner@example.invalid', 'test email')['port'] == 1025
        assert docker('exec', app, 'id', '-u') == '10001'
        docker('exec', app, 'mindflow', 'healthcheck')
        docker('restart', app)
        wait_for(lambda: request('/health/ready')[0] == {'ok': True})
        assert call(f'/pages/{page}/document')['document'] == document
        assert request('/api/v1/assets/' + asset['id'])[0] == original
        docker('stop', app)
        docker('run', '--rm', '-v', f'{volume}:/source:ro', '-v', f'{backup}:/target', 'alpine:3.23', 'cp', '-a', '/source/.', '/target/')
        docker('rm', app)
        docker('run', '--rm', '-v', f'{backup}:/data', args.image, 'check-data')
        docker('run', '--rm', '-v', f'{backup}:/data', args.image, 'revoke-credentials')
        start(backup)
        wait_for(lambda: request('/health/ready')[0] == {'ok': True})
        call('/auth/me', expected=401)
        login()
        assert call(f'/pages/{page}/document')['document'] == document
        assert request('/api/v1/assets/' + asset['id'])[0] == original
        print('PASS: production HTTPS cookies and SSE streaming, implicit TLS/STARTTLS email and invitation delivery, owner isolation, BLOBs/variants, save retries/takeover, non-root read-only runtime, restart persistence, snapshot restore and credential revocation')
finally:
    # These identifiers are unique to this invocation; never touch normal app resources.
    for container in (app, fixture):
        subprocess.run(['docker', 'rm', '-f', container], capture_output=True)
    for resource in (volume, backup):
        subprocess.run(['docker', 'volume', 'rm', resource], capture_output=True)
    subprocess.run(['docker', 'network', 'rm', network], capture_output=True)
