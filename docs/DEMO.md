# Recording the README demo

Run `make demo` from the repository root after installing frontend dependencies (`cd web && npm ci`). You need Go, Node.js 22.12+, Python 3.11+, and network access for the first browser/encoder installation.

The command builds the frontend and backend, ensures Chromium is installed, and runs `web/scripts/record-demo.mjs`. It creates a temporary SQLite database and binds a development instance only to `127.0.0.1:8089`. That port must be free. The temporary users are `alex@example.com` and `sam@example.com`; the SMTP target is a closed local port, so no real email is sent. The script consumes the invitation from that fixture's email queue and activates the demo collaborator before recording.

The visible workflow uses the actual UI: create a workspace, upload a generated landscape PNG, drag images/shapes/text to size, draw a stroke, share comment access, and post a comment. Captions and a visible pointer are added only to the recording page. Edit the script's steps, captions, or coordinates when the UI changes.

Playwright records a WebM video and timed PNG frames. `scripts/demo-gif.py` converts the frames to a 960px looping GIF with a shared palette, using [Pillow's GIF encoder](https://pillow.readthedocs.io/en/stable/handbook/image-file-formats.html#gif). Encoder dependencies are pinned in `scripts/demo-requirements.txt` and included in `make upgrade`.

Outputs:

- `docs/media/workflow.gif`: the README artifact to review and commit.
- `.cache/demo/video/`: raw WebM recordings for local review; not committed.

Temporary database and PNG-frame directories are removed when the script exits. It does not connect to an existing instance or publish anything. Do not replace its local origin with a production URL. Review the GIF after recording, and commit it alongside UI changes when appropriate. The recording is deliberately manual and not part of CI.
