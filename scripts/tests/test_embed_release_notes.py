import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


class ReleaseNotesTests(unittest.TestCase):
    def test_release_body_is_data_and_keeps_markdown(self):
        with tempfile.TemporaryDirectory() as directory:
            event = Path(directory) / 'event.json'
            output = Path(directory) / 'notes.json'
            body = '## New features\n- Faster images\n- Literal $(touch /tmp/not-executed) and `commands`.'
            event.write_text(json.dumps({'release': {'tag_name': 'v1.3.5', 'body': body}}))
            subprocess.run([sys.executable, 'scripts/embed-release-notes.py', '--event', str(event), '--output', str(output)], check=True)
            self.assertEqual(json.loads(output.read_text()), {'version': 'v1.3.5', 'body': body})

    def test_empty_release_body_is_supported(self):
        with tempfile.TemporaryDirectory() as directory:
            event = Path(directory) / 'event.json'
            output = Path(directory) / 'notes.json'
            event.write_text(json.dumps({'release': {'tag_name': 'v1.3.5', 'body': None}}))
            subprocess.run([sys.executable, 'scripts/embed-release-notes.py', '--event', str(event), '--output', str(output)], check=True)
            self.assertEqual(json.loads(output.read_text())['body'], '')
