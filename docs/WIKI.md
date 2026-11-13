# Publishing the GitHub wiki

`docs/` is the source of truth. The **Publish documentation to wiki** workflow runs on every push to `main`; it can also be started manually on `main` from the Actions tab. It uses the current main-branch checkout so a queued run does not publish an older snapshot.

The workflow clones the existing wiki, prepares the Markdown and assets, commits changed content, and pushes a normal commit. It does not force-push or rewrite history. Runs are serialized, and an unchanged publication makes no commit.

`scripts/prepare-wiki.py` generates a home page and sidebar, rewrites documentation links to wiki pages, and sends links outside `docs/` back to files on the repository's main branch. Code fences are left unchanged. Nested Markdown filenames become hyphenated page names. Assets retain their directory structure. If `docs/Home.md` or `docs/_Sidebar.md` exists, its content is used instead of the generated page.

The hidden `.mindflow-docs.json` manifest records managed files. Renamed/deleted source files disappear from the next publication; unrelated manually created wiki pages are preserved. Direct wiki edits to managed pages are replaced on the next sync, so edit those pages in `docs/` instead.

## One-time setup

Enable the repository wiki and create its first page in GitHub if it is still empty. Enabling the feature alone does not initialize its Git repository. See [GitHub's wiki instructions](https://docs.github.com/en/communities/documenting-your-project-with-wikis/adding-or-editing-wiki-pages).

The job requests `contents: write` and uses its automatic `GITHUB_TOKEN`; no extra repository secret is configured. Repository/organization policy must permit that write permission. Tokens are provided through a temporary Git askpass helper, rather than embedded into remote URLs or Git configuration. Checkout credentials are not persisted.

To preview the generated content without publishing:

```sh
python3 scripts/prepare-wiki.py \
  --source docs \
  --destination .cache/wiki-preview \
  --repository https://github.com/Neokil/mindflow
```
