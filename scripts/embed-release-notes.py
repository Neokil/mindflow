"""Capture release metadata at build time; running servers need no GitHub access."""
import argparse
import json
from pathlib import Path


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--event', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    release = json.loads(args.event.read_text())['release']
    version, body = release['tag_name'], release.get('body') or ''
    if not isinstance(version, str) or not isinstance(body, str):
        raise ValueError('Release tag and body must be strings.')
    if len(body.encode('utf-8')) > 256 * 1024:
        raise ValueError('Release notes exceed the 256 KiB limit.')
    args.output.write_text(json.dumps({'version': version, 'body': body}, ensure_ascii=False) + '\n')


if __name__ == '__main__':
    main()
