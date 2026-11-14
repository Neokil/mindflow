#!/usr/bin/env python3
"""Prepare repository docs for a wiki checkout without touching Git history."""
import argparse
import json
from pathlib import Path
import re
import shutil
from urllib.parse import quote, unquote, urlsplit, urlunsplit

MANIFEST = '.mindflow-docs.json'
LINK = re.compile(r'(!?\[[^\]\n]*\]\()([^\s)]+)([^\n)]*\))')


def rewrite_links(text, source_file, source, repository, pages, revision='main'):
    def replace(match):
        target = urlsplit(match[2])
        if target.scheme or target.netloc or not target.path:
            return match[0]
        path = (source_file.parent / unquote(target.path)).resolve()
        if path in pages:
            address = f'{repository}/wiki/{quote(pages[path])}'
        elif path.is_relative_to(source):
            # Assets retain their relative paths in the wiki checkout.
            address = quote(path.relative_to(source).as_posix())
        elif path.is_relative_to(source.parent):
            relative = quote(path.relative_to(source.parent).as_posix())
            kind = 'raw' if match[1].startswith('!') else 'blob'
            address = f'{repository}/{kind}/{quote(revision, safe="")}/{relative}'
        else:
            return match[0]
        parsed = urlsplit(address)
        return match[1] + urlunsplit((parsed.scheme, parsed.netloc, parsed.path, target.query, target.fragment)) + match[3]

    result = []
    fence = None
    for line in text.splitlines(keepends=True):
        marker = re.match(r'^\s*(`{3,}|~{3,})', line)
        if marker:
            current = marker[1]
            if fence is None:
                fence = current
            elif current[0] == fence[0] and len(current) >= len(fence):
                fence = None
            result.append(line)
        else:
            result.append(line if fence else LINK.sub(replace, line))
    return ''.join(result)


def safe_target(destination, name):
    relative = Path(name)
    target = destination / relative
    if relative.is_absolute() or '..' in relative.parts or '.git' in relative.parts or not target.resolve().is_relative_to(destination):
        raise ValueError(f'Unsafe wiki path: {name}')
    return target


def prepare(source, destination, repository, revision='main', version=None):
    source, destination = source.resolve(), destination.resolve()
    if source == destination or destination.is_relative_to(source) or source.is_relative_to(destination):
        raise ValueError('Source and wiki destination must be separate directories')
    files = sorted(path for path in source.rglob('*') if path.is_file())
    if any(path.is_symlink() for path in source.rglob('*')):
        raise ValueError('Docs must not contain symbolic links')
    pages = {path: path.relative_to(source).with_suffix('').as_posix().replace('/', '-') for path in files if path.suffix.lower() == '.md'}
    if len(set(pages.values())) != len(pages):
        raise ValueError('Nested docs produce duplicate wiki page names')
    generated = {path: f'{pages[path]}.md' if path in pages else path.relative_to(source).as_posix() for path in files}
    names = set(generated.values()) | {'Home.md', '_Sidebar.md'}
    manifest = safe_target(destination, MANIFEST)
    previous = json.loads(manifest.read_text()) if manifest.exists() else []
    # Validate every write/deletion before mutating the checkout, including old manifest entries.
    for name in names | set(previous):
        safe_target(destination, name)
    destination.mkdir(parents=True, exist_ok=True)
    for name in set(previous) - names:
        safe_target(destination, name).unlink(missing_ok=True)
    for path, name in generated.items():
        target = safe_target(destination, name)
        target.parent.mkdir(parents=True, exist_ok=True)
        if path in pages:
            target.write_text(rewrite_links(path.read_text(), path, source, repository, pages, revision))
        else:
            shutil.copyfile(path, target)
    navigation = '\n'.join(f'- [{title.replace("_", " ").replace("-", " ").title()}]({repository}/wiki/{quote(title)})' for title in pages.values() if title not in ('Home', '_Sidebar'))
    if 'Home.md' not in generated.values():
        (destination / 'Home.md').write_text(f'# mindflow documentation\n\nPublished from [`docs/`]({repository}/tree/{quote(revision, safe="")}/docs). Edit the source files in the main repository; wiki copies are updated automatically.\n\n{navigation}\n')
    if '_Sidebar.md' not in generated.values():
        (destination / '_Sidebar.md').write_text(f'[Home]({repository}/wiki/Home)\n\n{navigation}\n')
    home = destination / 'Home.md'
    text = home.read_text()
    label = re.sub(r'([\\`*_[\]<>])', r'\\\1', ' '.join((version or revision).split()))
    notice = f'> Documentation version: **{label}** · [View source]({repository}/tree/{quote(revision, safe="")}/docs).\n\n'
    heading = re.match(r'\A#[ \t]+[^\n]*(?:\n|$)\n*', text)
    position = heading.end() if heading else 0
    home.write_text(text[:position].rstrip('\n') + ('\n\n' if position else '') + notice + text[position:])
    (destination / MANIFEST).write_text(json.dumps(sorted(names), indent=2) + '\n')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', type=Path, default=Path('docs'))
    parser.add_argument('--destination', type=Path, required=True)
    parser.add_argument('--repository', required=True, help='Repository web URL')
    parser.add_argument('--revision', default='main', help='Repository revision used for source-file links')
    parser.add_argument('--version', help='Human-readable documentation version; defaults to the revision')
    options = parser.parse_args()
    prepare(options.source, options.destination, options.repository.rstrip('/'), options.revision, options.version)


if __name__ == '__main__':
    main()
