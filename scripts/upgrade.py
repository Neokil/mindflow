#!/usr/bin/env python3
"""Refresh stable dependencies and immutable build references, then verify the result."""
import argparse
import json
import os
from pathlib import Path
import re
import subprocess
import urllib.error
import urllib.parse
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
IGNORED = {'.git', '.cache', 'node_modules', 'dist', 'test-results', '__pycache__'}


def fetch(url, headers=None):
    request_headers = {'User-Agent': 'mindflow-upgrade', **(headers or {})}
    if url.startswith('https://api.github.com/') and os.environ.get('GITHUB_TOKEN'):
        request_headers['Authorization'] = 'Bearer ' + os.environ['GITHUB_TOKEN']
    request = urllib.request.Request(url, headers=request_headers)
    for attempt in range(3):
        try:
            with urllib.request.urlopen(request, timeout=30) as response:
                return json.load(response), response.headers
        except (TimeoutError, urllib.error.URLError):
            if attempt == 2:
                raise
    raise RuntimeError('Registry request did not complete')


def latest_release(repository):
    release, _ = fetch(f'https://api.github.com/repos/{repository}/releases/latest')
    return release['tag_name']


def action_pin(repository):
    tag = latest_release(repository)
    commit, _ = fetch(f'https://api.github.com/repos/{repository}/commits/{urllib.parse.quote(tag, safe="")}')
    sha = commit['sha']
    if not re.fullmatch(r'[0-9a-f]{40}', sha):
        raise ValueError(f'Invalid commit returned for {repository}')
    return sha, tag


def stable_image_tag(repository, flavor):
    candidates = []
    pattern = re.compile(r'^(\d+)\.(\d+)\.(\d+)' + re.escape(flavor) + '$')
    url = f'https://hub.docker.com/v2/repositories/library/{repository}/tags?page_size=100'
    # Rebuilt aliases are mixed into the tag feed. Compare stable semantic versions.
    for _ in range(5):
        data, _ = fetch(url)
        for entry in data['results']:
            match = pattern.fullmatch(entry['name'])
            if match:
                candidates.append((tuple(map(int, match.groups())), entry['name']))
        url = data.get('next')
        if not url:
            break
    if not candidates:
        raise ValueError(f'No stable Docker tag found for {repository}{flavor}')
    return max(candidates)[1]


def image_pin(repository, flavor):
    tag = stable_image_tag(repository, flavor)
    token, _ = fetch(f'https://auth.docker.io/token?service=registry.docker.io&scope=repository:library/{repository}:pull')
    _, headers = fetch(f'https://registry-1.docker.io/v2/library/{repository}/manifests/{tag}', {
        'Authorization': 'Bearer ' + token['token'],
        'Accept': 'application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json',
    })
    digest = headers['Docker-Content-Digest']
    if not re.fullmatch(r'sha256:[0-9a-f]{64}', digest):
        raise ValueError(f'Invalid Docker digest for {repository}:{tag}')
    return f'{repository}:{tag}@{digest}'


def source_files():
    for directory, children, files in os.walk(ROOT):
        children[:] = [child for child in children if child not in IGNORED]
        for name in files:
            path = Path(directory, name)
            if name == 'Dockerfile' or name.endswith('.Dockerfile') or path == ROOT / 'scripts/verify-container.py':
                yield path


def plan():
    changes = {}
    images = {}
    image_reference = re.compile(r'\b(node|golang|alpine|python):[\w.\-]+(?:@sha256:[0-9a-f]{64})?')
    for path in source_files():
        source = path.read_text()
        def replace_image(match):
            repository = match[1]
            flavor = '-alpine' if '-alpine' in match[0] else ''
            key = repository, flavor
            if key not in images:
                images[key] = image_pin(*key)
            return images[key]
        changes[path] = image_reference.sub(replace_image, source)
    node_version = images[('node', '-alpine')].split(':', 1)[1].split('-alpine')[0]
    go_version = images[('golang', '-alpine')].split(':', 1)[1].split('-alpine')[0]
    changes[ROOT / 'go.mod'] = re.sub(r'(?m)^go \S+$', 'go ' + go_version, (ROOT / 'go.mod').read_text())
    actions = {}
    for path in sorted((ROOT / '.github/workflows').glob('*.y*ml')):
        def replace_action(match):
            repository, subpath = match[1], match[2] or ''
            if repository not in actions:
                actions[repository] = action_pin(repository)
            sha, tag = actions[repository]
            return f'uses: {repository}{subpath}@{sha} # {tag}'
        source = re.sub(r'uses:\s+([\w-]+/[\w.-]+)(/[\w/.-]+)?@[^\s]+(?:\s+# [^\n]+)?', replace_action, path.read_text())
        changes[path] = re.sub(r'node-version:\s*[\d.]+', 'node-version: ' + node_version, source)
    package_path = ROOT / 'web/package.json'
    package = json.loads(package_path.read_text())
    def upgrade_npm(entries):
        for name, value in entries.items():
            if isinstance(value, dict):
                upgrade_npm(value)
            else:
                metadata, _ = fetch('https://registry.npmjs.org/' + urllib.parse.quote(name, safe='') + '/latest')
                entries[name] = metadata['version']
    for section in ('dependencies', 'devDependencies', 'overrides'):
        upgrade_npm(package.get(section, {}))
    changes[package_path] = json.dumps(package, indent=2) + '\n'
    makefile = (ROOT / 'Makefile').read_text()
    for variable, repository in {'SQLC_VERSION': 'sqlc-dev/sqlc', 'GOLANGCI_VERSION': 'golangci/golangci-lint', 'GOVULNCHECK_VERSION': 'golang/vuln'}.items():
        # Go vulnerability tooling uses Go module releases rather than GitHub release assets.
        if variable == 'GOVULNCHECK_VERSION':
            latest, _ = fetch('https://proxy.golang.org/golang.org/x/vuln/@latest')
            version = latest['Version']
        else:
            version = latest_release(repository)
        makefile = re.sub(rf'(?m)^{variable} := .*$', variable + ' := ' + version, makefile)
    changes[ROOT / 'Makefile'] = makefile
    return changes


def run(*args, directory=ROOT):
    print('+', ' '.join(args), flush=True)
    subprocess.run(args, cwd=directory, check=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--dry-run', action='store_true', help='resolve and show updates without changing files or installing anything')
    parser.add_argument('--full', action='store_true', help='also run browser and Docker verification')
    options = parser.parse_args()
    # Resolve the complete build/reference plan before modifying any source files.
    changes = plan()
    for path, source in changes.items():
        if source != path.read_text():
            print('Update', path.relative_to(ROOT))
            if not options.dry_run:
                path.write_text(source)
    if options.dry_run:
        return
    run('go', 'get', '-u', './...')
    run('go', 'mod', 'tidy')
    run('npm', 'install', directory=ROOT / 'web')
    run('make', 'generate')
    run('make', 'lint', 'test', 'audit')
    if options.full:
        run('make', 'test-e2e', 'verify-container')
    print('Review the diff and runtime license notices before committing. No commit, push, or publication was performed.')


if __name__ == '__main__':
    main()
