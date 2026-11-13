# Recording the README demo

Run `make demo` from the repository root after installing frontend dependencies (`cd web && npm ci`). You need Go, Node.js 22.12+, Python 3.11+, and network access for the first browser/encoder installation.

The command builds the frontend and backend, ensures Chromium is installed, and runs `web/scripts/record-demo.mjs`. It creates a temporary SQLite database and binds a development instance only to `127.0.0.1:8089`. That port must be free. The temporary users are `alex@example.com` and `sam@example.com`; the SMTP target is a closed local port, so no real email is sent. The script consumes the invitation from that fixture's email queue and activates the demo collaborator before recording.

The browser emulates a dark system appearance, and the application follows its default system preference. The GIF and captions are recorded in dark mode.

The visible workflow uses the actual UI: create a workspace, upload a generated landscape PNG, drag images/shapes/text to size, draw a stroke, share comment access, and post a comment. Captions and a visible pointer are added only to the recording page. Edit the script's steps, captions, or coordinates when the UI changes.

Playwright records a WebM video and timed PNG frames. `scripts/demo-gif.py` converts the frames to a 960px looping GIF with a shared palette, using [Pillow's GIF encoder](https://pillow.readthedocs.io/en/stable/handbook/image-file-formats.html#gif). Encoder dependencies are pinned in `scripts/demo-requirements.txt` and included in `make upgrade`.

Outputs:

- `docs/media/workflow.gif`: the README artifact to review and commit.
- `.cache/demo/video/`: raw WebM recordings for local review; not committed.

Temporary database and PNG-frame directories are removed when the script exits. It does not connect to an existing instance or publish anything. Do not replace its local origin with a production URL. Review the GIF after recording, and commit it alongside UI changes when appropriate. The recording is deliberately manual and not part of CI.


## Recording a social announcement reel

Run `make reel` to record an isolated local demo and render a **22-second, 1080×1920, 9:16, 30 fps H.264 MP4**. It uses real app captures, the current icon and brand palette, large baked-in captions, and an invitation to DM the author for testing access. The video is silent so it can be posted as-is or paired with music in the posting app. This command is manual and does not publish anything.

The fixture binds only to `127.0.0.1:8091`, uses a temporary database and synthetic `.invalid` accounts, and sends no real email. Its initial board is seeded through the normal authenticated APIs; image placement, drawing, sharing, and commenting are then recorded through the UI. The temporary database is removed on exit. Never change the recorder to use a deployed instance.

Outputs stay out of Git under `.cache/reel/export/`:

- `mindflow-announcement.mp4`: ready-to-post reel.
- `mindflow-cover.jpg`: portrait cover image.
- `caption.txt`: suggested accompanying post.
- `mindflow-announcement-kit.zip`: video, cover, caption, and usage notes together.
- `storyboard.jpg` and `preview-*.jpg`: review images for each section.

Edit `web/scripts/record-reel.mjs` for the app workflow and `scripts/render-reel.py` for copy, timing, colors, and composition. Raw captures and their timing manifest remain under `.cache/reel/`; rerender without recording by running `.cache/reel-venv/bin/python scripts/render-reel.py`. The renderer uses macOS system fonts or DejaVu Sans on Linux; `--font` and `--bold-font` accept alternative font paths. Pillow and the bundled FFmpeg encoder are pinned in `scripts/reel-requirements.txt` and updated by `make upgrade`.

Review the storyboard and watch the MP4 before posting. Physical iPad footage is not claimed: these are recordings of the actual web app in desktop Chromium. The reel makes no claim of public registration; access is offered by contacting the author.
