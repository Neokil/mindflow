# Branding and colors

The supplied artwork defines mindflow's forest palette. Keep the interface quiet and use color to clarify actions, selection, and focus.

| Color | Hex | Role |
| --- | --- | --- |
| Forest | `#1B5E3B` | Primary actions, links, and focus in light mode; text on Sprout buttons in dark mode; email branding. |
| Leaf | `#4CAF50` | Supporting illustration details and decorative accents. |
| Sprout | `#A4D65E` | Primary actions, links, focus, and selection in dark mode; supporting illustrations. |
| Gold | `#D9A441` | Sparing warm highlights and warning accents; use a darker Gold-derived tone for warning text on light backgrounds. |

CSS defines these as `--forest`, `--leaf`, `--sprout`, and `--gold`. Components normally use semantic `--accent` and `--accent-soft` so appearance changes do not require component-specific palettes. Soft surfaces use green/gold tints. Neutral surfaces remain restrained; error/destructive states keep their semantic red.

Primary buttons use white on Forest in light mode (7.75:1 contrast) and Forest on Sprout in dark mode (4.56:1). Leaf and Gold are supporting colors rather than small text on white, where their contrast is insufficient. Focus rings and canvas selection guides use the active theme accent.

## Assets

- [`web/public/icon.png`](../../web/public/icon.png) is the application mark, used in headers, authentication screens, and the favicon. Preserve its transparency and proportions; do not stretch or recolor it.
- [`web/public/banner.png`](../../web/public/banner.png) is the README banner. Display it proportionally and keep its existing background/artwork.
- [`docs/media/workflow.gif`](../media/workflow.gif) shows the app in dark mode. Regenerate it with `make demo` after branding changes; see [demo recording](demo.md).

The old `logo.png` asset has been replaced by the new icon. The reusable `Brand` component owns the application mark and wordmark; use it instead of adding individual logo references throughout the app.

## Canvas and email

Konva canvas paint styles cannot resolve CSS variables. `useCanvasAccent` reads the computed `--accent` and subscribes to changes of the root `data-theme` attribute. Pass that resolved color to selection outlines, transform handles, and insertion/lasso previews.

New drawing tools start with the current theme accent and a pale Sprout-derived fill. User-chosen colors and stored document elements remain content: switching appearance does not recolor them. Imported colors and uploaded image pixels are preserved.

HTML emails use Forest for their heading, action button, and fallback link, with white button text. They retain escaped content, clickable links, and a plain-text alternative. Remote logo images are not required to read credential emails.

Validate branding in light and dark system appearance. The appearance browser test checks the loaded icon, favicon, button contrast pairings, and live theme switching in Chrome and Firefox. Review the README banner and refreshed demo GIF visually.
