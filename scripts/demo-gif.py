#!/usr/bin/env python3
"""Encode the Playwright demo's timed PNG frames as a compact looping GIF."""
import json
from pathlib import Path
import sys
from PIL import Image

manifest = Path(sys.argv[1])
target = Path(sys.argv[2])
entries = json.loads(manifest.read_text())
frames = []
for entry in entries:
    with Image.open(manifest.parent / entry['file']) as source:
        # Recorded UI only: no external or user-supplied image decoder input.
        frame = source.convert('RGB')
        frame.thumbnail((960, 640), Image.Resampling.LANCZOS)
        frames.append(frame)
# One palette avoids flashes between frames while preserving readable UI text.
preview = Image.new('RGB', (frames[0].width, frames[0].height * 8), 'white')
for index in range(8):
    preview.paste(frames[min(len(frames) - 1, index * len(frames) // 8)], (0, index * frames[0].height))
palette = preview.quantize(colors=192)
indexed = [frame.quantize(palette=palette, dither=Image.Dither.NONE) for frame in frames]
target.parent.mkdir(parents=True, exist_ok=True)
indexed[0].save(target, save_all=True, append_images=indexed[1:], duration=[entry['duration'] for entry in entries], loop=0, optimize=True, disposal=1)
print(f'Created {target}: {len(frames)} frames, {target.stat().st_size / 1024:.0f} KiB')
