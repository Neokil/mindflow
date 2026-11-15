# Manual interoperability checklist

Use disposable workspaces. Keep original notebooks, and compare content before discarding any source. Automated tests verify parser/API/browser round trips; they do not launch Goodnotes, OneNote, or Butterfly themselves.

## Prepare one reference workspace

Create two pages. Add text with bold/italic, a list, a pressure-varying pen stroke, a highlight, a rectangle, a circle, a star, an image, a cropped/rotated image, and an embedded PDF. Add a group, a link, and a comment to verify the documented exclusions. Move some content into negative coordinates. Give the second page a different background color.

## mindflow and PDF baseline

- [ ] Export the workspace as `.mindflow`, reimport, and compare both pages, media, pressure, groups, and included links.
- [ ] Export only page two and append it to another workspace; unrelated pages/assets should be absent.
- [ ] Export workspace PDF: two PDF pages, all content fitted, no app chrome. Open it in a separate PDF reader.
- [ ] Reimport that PDF and annotate the resulting page images. Also try the editor's single-canvas and embedded-viewer options.
- [ ] Export workspace PNG: two numbered images in a ZIP. Export one page: one PNG.

## Butterfly

- [ ] Export `.bfly` and open it in Butterfly. Check both page names/order, shapes, text, images, rotations/crops, and strokes.
- [ ] Edit a text box and a stroke in Butterfly, export again, and import into mindflow.
- [ ] Compare pressure/highlights and typography against the conversion notes. Stars may be native pen paths rather than star controls.

## Goodnotes

- [ ] Import your original large Goodnotes test notebook locally. Compare text, images, PDF-derived content, and ink with the original. Check continuation pages and conversion notes for this large source file.
- [ ] Export the reference workspace as `.goodnotes` and open it in Goodnotes. **App acceptance of the generated native format is a required compatibility check.**
- [ ] Edit an ordinary text box and select/move a stroke independently. Shapes/rotated text are individual images; verify they do not flatten the whole page.
- [ ] Check image crops/rotation, backgrounds, page names/order, text alignment, scale, and pen/highlighter appearance.
- [ ] Export the edited copy from Goodnotes and reimport it into mindflow.
- [ ] Optionally open `docs/examples/mindflow-compatibility.goodnotes`. It contains only generated text, a green stroke, and a green/gold image; it is a format probe, not evidence of app compatibility.

## OneNote

- [ ] Import your original OneNote test notebook locally. For the provided example, expect 14 active pages; compare section prefixes, text, and ink.
- [ ] Export both `.onepkg` and `.one` from the reference workspace. Open them in desktop OneNote, then check mobile/tablet access through normal OneNote sync if needed.
- [ ] Edit text, select/move ink, and move an image independently. Check shape/rotated-text PNG fallbacks and conversion notes.
- [ ] Export the edited copy from OneNote and reimport it. Compare page order, pressure, and layout.

## Bulk migration and screenshots

- [ ] Select Goodnotes, OneNote, Butterfly, PDF, and mindflow files together, plus one intentionally broken file. Successful files must remain available; retry must not duplicate them.
- [ ] Append Goodnotes, OneNote, PDF, and mindflow pages to an existing workspace. A shared read-only user must not append.
- [ ] Screenshot a small region, then another at a different zoom. Inspect dimensions/content in the PNG and paste the clipboard image into another application over HTTPS.
- [ ] Repeat region selection with touch/Pencil on iPad. It must not pan the browser or modify the canvas.
- [ ] Verify the clipboard explanation on plain HTTP and a failed/denied clipboard permission.
- [ ] Verify a user without workspace access cannot export its content or media.

Private sample files are not committed or uploaded by the test suite. Record the destination app/version and any visual or editing differences when reporting compatibility failures.
