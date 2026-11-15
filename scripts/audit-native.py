#!/usr/bin/env python3
"""Fail CI if the native adapter's locked dependency graph has known advisories."""
import json
import os
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[1]
environment = {**os.environ, 'DOTNET_CLI_HOME': str(ROOT / '.cache/dotnet-home'),
               'NUGET_PACKAGES': str(ROOT / '.cache/nuget')}
result = subprocess.run(['dotnet', 'list', str(ROOT / 'internal/workspace/onenote/codec/OneNoteCodec.csproj'),
                         'package', '--vulnerable', '--include-transitive', '--format', 'json'],
                        check=True, capture_output=True, text=True, env=environment)
report = json.loads(result.stdout)
issues = []
for project in report.get('projects', []):
    for framework in project.get('frameworks', []):
        for package in framework.get('topLevelPackages', []) + framework.get('transitivePackages', []):
            if package.get('vulnerabilities'):
                issues.append(package['id'])
if issues:
    raise SystemExit('Native dependency vulnerabilities: ' + ', '.join(sorted(set(issues))))
print('Native dependency audit: no known vulnerabilities in the locked graph.')
