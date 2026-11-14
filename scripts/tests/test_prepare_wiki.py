import importlib.util
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('prepare_wiki', Path(__file__).parents[1] / 'prepare-wiki.py')
wiki = importlib.util.module_from_spec(spec)
spec.loader.exec_module(wiki)


class WikiTests(unittest.TestCase):
    def test_links_assets_navigation_and_fenced_examples(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            docs, target = root / 'repo/docs', root / 'wiki'
            (docs / 'guides').mkdir(parents=True)
            (docs / 'media').mkdir()
            (docs / 'guides/start.md').write_text('# Start\n![Demo](../media/demo.gif)\n')
            (docs / 'media/demo.gif').write_bytes(b'GIF89a')
            (docs / 'Index.md').write_text('[Start](guides/start.md#install) [Source](../README.md) [Web](https://example.com)\n```md\n[Literal](guides/start.md)\n```\n')
            wiki.prepare(docs, target, 'https://github.com/owner/repo')
            text = (target / 'Index.md').read_text()
            self.assertIn('(https://github.com/owner/repo/wiki/guides-start#install)', text)
            self.assertIn('(https://github.com/owner/repo/blob/main/README.md)', text)
            self.assertIn('[Literal](guides/start.md)', text)
            self.assertIn('[Web](https://example.com)', text)
            self.assertEqual((target / 'media/demo.gif').read_bytes(), b'GIF89a')
            self.assertIn('![Demo](media/demo.gif)', (target / 'guides-start.md').read_text())
            self.assertIn('wiki/guides-start', (target / '_Sidebar.md').read_text())
            self.assertIn('docs/', (target / 'Home.md').read_text())

    def test_release_revision_pins_repository_links_and_generated_home(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            docs, target = root / 'repo/docs', root / 'wiki'
            docs.mkdir(parents=True)
            (docs / 'Guide.md').write_text('[Source](../README.md#install) ![Icon](../web/public/icon.png)\n')
            repository = 'https://github.com/owner/repo'
            for revision, encoded in [('v1.2.0', 'v1.2.0'), ('release/next', 'release%2Fnext')]:
                with self.subTest(revision=revision):
                    wiki.prepare(docs, target, repository, revision)
                    text = (target / 'Guide.md').read_text()
                    self.assertIn(f'{repository}/blob/{encoded}/README.md#install', text)
                    self.assertIn(f'{repository}/raw/{encoded}/web/public/icon.png', text)
                    self.assertIn(f'{repository}/tree/{encoded}/docs', (target / 'Home.md').read_text())

    def test_deletes_only_previous_managed_files_and_is_repeatable(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            docs, target = root / 'docs', root / 'wiki'
            docs.mkdir()
            (docs / 'Old.md').write_text('old')
            wiki.prepare(docs, target, 'https://github.com/owner/repo')
            (target / 'Manual.md').write_text('keep')
            (docs / 'Old.md').rename(docs / 'New.md')
            wiki.prepare(docs, target, 'https://github.com/owner/repo')
            before = {path.name: path.read_bytes() for path in target.iterdir()}
            wiki.prepare(docs, target, 'https://github.com/owner/repo')
            self.assertEqual(before, {path.name: path.read_bytes() for path in target.iterdir()})
            self.assertFalse((target / 'Old.md').exists())
            self.assertEqual((target / 'Manual.md').read_text(), 'keep')

    def test_authored_home_and_grouped_sidebar_use_nested_wiki_links(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            docs, target = root / 'docs', root / 'wiki'
            (docs / 'product').mkdir(parents=True)
            (docs / 'product/overview.md').write_text('# Product definition\n')
            (docs / 'Home.md').write_text('# Documentation\n[Definition](product/overview.md)\n')
            (docs / '_Sidebar.md').write_text('### Product\n[Definition](product/overview.md)\n')
            wiki.prepare(docs, target, 'https://github.com/owner/repo')
            self.assertTrue((target / 'Home.md').read_text().startswith('# Documentation'))
            sidebar = (target / '_Sidebar.md').read_text()
            self.assertIn('### Product', sidebar)
            self.assertIn('(https://github.com/owner/repo/wiki/product-overview)', sidebar)
            self.assertIn('wiki/product-overview', (target / 'Home.md').read_text())

    def test_manifest_cannot_delete_outside_wiki(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            docs, target = root / 'docs', root / 'wiki'
            docs.mkdir()
            target.mkdir()
            (root / 'keep.txt').write_text('keep')
            (target / wiki.MANIFEST).write_text('["../keep.txt"]')
            with self.assertRaises(ValueError):
                wiki.prepare(docs, target, 'https://github.com/owner/repo')
            self.assertTrue((root / 'keep.txt').exists())


if __name__ == '__main__':
    unittest.main()
