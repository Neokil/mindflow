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
