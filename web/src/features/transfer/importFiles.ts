import { api } from "../../api";
import { emptyDocument, uid } from "../../model";
import { MAX_PDF_BYTES, openPDF, renderPDFPage } from "../pdf/pdf";
export type ImportResult = {
  id: string;
  pageId: string;
  pageCount: number;
  elementCount: number;
  warnings: string[];
};
export const IMPORT_EXTENSIONS = ".mindflow,.bfly,.pdf,.goodnotes";
export function importName(file: File) {
  return (
    file.name
      .replace(/\.(mindflow|bfly|pdf|goodnotes)$/i, "")
      .trim()
      .slice(0, 180) || "Imported workspace"
  );
}
export async function importFile(
  file: File,
  name: string,
  workspaceId: string | undefined,
  progress: (message: string) => void,
): Promise<ImportResult> {
  const extension = file.name.split(".").pop()?.toLowerCase();
  if (!["mindflow", "bfly", "pdf", "goodnotes"].includes(extension || ""))
    throw new Error("Choose a mindflow, Butterfly, Goodnotes, or PDF file.");
  if (file.size > (extension === "pdf" ? MAX_PDF_BYTES : 64 * 1024 * 1024))
    throw new Error(
      "File exceeds the import size limit (PDF: 50 MiB; other files: 64 MiB).",
    );
  if (workspaceId && extension === "bfly")
    throw new Error(
      "Import Butterfly files as new workspaces from the overview.",
    );
  const form = new FormData();
  let warnings: string[] = [];
  if (extension === "goodnotes") {
    const converted = await (
      await import("./goodnotes/convert")
    ).importGoodnotes(file, name, progress);
    warnings = converted.warnings;
    form.append("file", converted.blob, "import.mindflow");
  } else if (extension === "pdf")
    form.append(
      "file",
      await pdfBundle(file, name, progress),
      "import.mindflow",
    );
  else form.append("file", file);
  form.append("name", name);
  progress("Saving imported workspace…");
  const result = await api<ImportResult>(
    workspaceId
      ? `/workspaces/${workspaceId}/import/mindflow`
      : `/workspaces/import/${extension === "bfly" ? "butterfly" : "mindflow"}`,
    "POST",
    form,
  );
  return { ...result, warnings: [...warnings, ...result.warnings] };
}
async function pdfBundle(
  file: File,
  name: string,
  progress: (message: string) => void,
): Promise<Blob> {
  const controller = new AbortController();
  const pdf = await openPDF(
    new Uint8Array(await file.arrayBuffer()),
    controller.signal,
  );
  const pages = [],
    assets = [],
    files: Record<string, Uint8Array> = {};
  let bytes = 0;
  try {
    for (let i = 1; i <= pdf.numPages; i++) {
      progress(`Rendering PDF page ${i} of ${pdf.numPages}…`);
      const rendered = await renderPDFPage(pdf, i, controller.signal);
      const id = `a${i}`,
        document = emptyDocument();
      document.settings.background = "plain";
      document.settings.backgroundColor = "#ffffff";
      document.elements = [
        {
          id: uid(),
          type: "image",
          assetId: id,
          x: rendered.width / 2,
          y: rendered.height / 2,
          width: rendered.width,
          height: rendered.height,
          rotation: 0,
        },
      ];
      pages.push({ id: `p${i}`, name: `Page ${i}`, document });
      assets.push({ id, mime: "image/png" });
      bytes += rendered.blob.size;
      if (bytes > 60 * 1024 * 1024)
        throw new Error(
          "Rendered PDF exceeds the 64 MiB transfer limit. Split it into smaller documents.",
        );
      files[`assets/${id}.bin`] = new Uint8Array(
        await rendered.blob.arrayBuffer(),
      );
    }
    files["manifest.json"] = new TextEncoder().encode(
      JSON.stringify({
        format: "mindflow",
        version: 1,
        kind: "workspace",
        name,
        pages,
        templates: [],
        assets,
      }),
    );
    const { zipSync } = await import("fflate");
    const data = zipSync(files, { level: 0 });
    if (data.byteLength > 64 * 1024 * 1024)
      throw new Error(
        "Converted PDF exceeds 64 MiB. Split it into smaller documents.",
      );
    return new Blob([new Uint8Array(data)], {
      type: "application/vnd.mindflow+zip",
    });
  } finally {
    await pdf.loadingTask.destroy();
  }
}
