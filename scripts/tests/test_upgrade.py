import importlib.util
from pathlib import Path
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('upgrade', Path(__file__).parents[1] / 'upgrade.py')
upgrade = importlib.util.module_from_spec(spec)
spec.loader.exec_module(upgrade)


class UpgradeTests(unittest.TestCase):
    def test_stable_tags_exclude_prereleases_and_choose_semantic_order(self):
        feed = {'results': [{'name': tag} for tag in ['26.9.0-alpine', '26.10.0-alpine', '27.0.0-rc.1-alpine', 'latest']], 'next': None}
        with patch.object(upgrade, 'fetch', return_value=(feed, {})):
            self.assertEqual(upgrade.stable_image_tag('node', '-alpine'), '26.10.0-alpine')

    def test_action_tags_are_resolved_to_immutable_commits(self):
        with patch.object(upgrade, 'latest_release', return_value='v9.2.1'), patch.object(upgrade, 'fetch', return_value=({'sha': 'a' * 40}, {})):
            self.assertEqual(upgrade.action_pin('actions/checkout'), ('a' * 40, 'v9.2.1'))

    def test_invalid_registry_digests_are_rejected(self):
        with patch.object(upgrade, 'stable_image_tag', return_value='3.24.1'), patch.object(upgrade, 'fetch', side_effect=[({'token': 'fixture'}, {}), ({}, {'Docker-Content-Digest': 'unsafe'})]):
            with self.assertRaises(ValueError):
                upgrade.image_pin('alpine', '')


if __name__ == '__main__':
    unittest.main()

class NativeUpgradeTests(unittest.TestCase):
    def test_codec_updates_sdk_checksums_runtime_and_package_together(self):
        metadata = {'latest-sdk': '10.0.999', 'latest-release': '10.0.99', 'releases': [{
            'release-version': '10.0.99', 'runtime': {'version': '10.0.99'},
            'sdk': {'version': '10.0.999', 'files': [
                {'rid': 'linux-musl-x64', 'hash': 'a' * 128},
                {'rid': 'linux-musl-arm64', 'hash': 'b' * 128},
            ]},
        }]}
        dockerfile = upgrade.ROOT / 'Dockerfile'
        workflow = upgrade.ROOT / '.github/workflows/quality.yml'
        changes = {dockerfile: dockerfile.read_text(), workflow: workflow.read_text()}
        with patch.object(upgrade, 'fetch', side_effect=[(metadata, {}), ({'versions': ['3.6.0', '4.0.0-beta.1', '3.10.0']}, {})]):
            upgrade.native_updates(changes)
        self.assertIn('/Sdk/10.0.999/dotnet-sdk-10.0.999-', changes[dockerfile])
        self.assertIn('rid=x64; checksum=' + 'a' * 128, changes[dockerfile])
        self.assertIn('rid=arm64; checksum=' + 'b' * 128, changes[dockerfile])
        self.assertIn('dotnet-version: 10.0.999', changes[workflow])
        project = changes[upgrade.ROOT / 'internal/workspace/onenote/codec/OneNoteCodec.csproj']
        self.assertIn('<RuntimeFrameworkVersion>10.0.99</RuntimeFrameworkVersion>', project)
        self.assertIn('Version="3.10.0"', project)
