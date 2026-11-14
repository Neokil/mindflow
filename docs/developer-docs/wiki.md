# Publishing the GitHub wiki

`docs/` is the source of truth. The **Publish documentation to wiki** workflow is reusable and runs from the release workflow after the Docker image has been published successfully. It publishes docs from the release commit, so the wiki matches the shipped version. Branch and tag pushes do not publish the wiki. You can also start the wiki workflow manually from the Actions tab and select the branch or tag to publish.

The workflow clones the existing wiki, prepares the Markdown and assets, commits changed content, and pushes a normal commit. It does not force-push or rewrite history. Runs are serialized, and an unchanged publication makes no commit.

`scripts/prepare-wiki.py` generates a home page and sidebar, rewrites documentation links to wiki pages, and sends links outside `docs/` back to files at the checked-out commit. The workflow supplies `--revision` with the commit SHA; local previews default to `main`. Code fences are left unchanged. The [documentation home](../Home.md) and [sidebar](../_Sidebar.md) group pages into Product, User Docs, and Developer Docs. Nested Markdown filenames become hyphenated wiki page names, while their source files remain organized in directories. Assets retain their directory structure. If `docs/Home.md` or `docs/_Sidebar.md` exists, its content is used instead of the generated page.

When adding or moving a guide, update `docs/Home.md` and `docs/_Sidebar.md`. Use relative Markdown links; the publisher converts them for the wiki.

The hidden `.mindflow-docs.json` manifest records managed files. Renamed/deleted source files disappear from the next publication; unrelated manually created wiki pages are preserved. Direct wiki edits to managed pages are replaced on the next sync, so edit those pages in `docs/` instead.

## One-time setup

Enable the repository wiki and create its first page in GitHub if it is still empty. Enabling the feature alone does not initialize its Git repository. See [GitHub's wiki instructions](https://docs.github.com/en/communities/documenting-your-project-with-wikis/adding-or-editing-wiki-pages).

The calling release job grants `contents: write` only to wiki publishing; its check and image jobs retain read access. The wiki job requests `contents: write` and uses its automatic `GITHUB_TOKEN`; no extra repository secret is configured. Repository/organization policy must permit that write permission. Tokens are provided through a temporary Git askpass helper, rather than embedded into remote URLs or Git configuration. Checkout credentials are not persisted.

To preview the generated content without publishing:

```sh
python3 scripts/prepare-wiki.py \
  --source docs \
  --destination .cache/wiki-preview \
  --repository https://github.com/Neokil/mindflow \
  --revision v1.2.0
```
