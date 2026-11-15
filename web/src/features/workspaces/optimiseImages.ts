import { api } from "../../api";
import { clearMedia } from "../../media";
import type { Page } from "../../model";
import type { ImportResult } from "../transfer/importFiles";
import { refreshWorkspacePreview } from "./preview";

export type OptimisationResult = { optimised: number; images: number };
export async function optimisePageImages(pageId: string, zoom = 1) {
  const result = await api<OptimisationResult>(
    `/pages/${pageId}/optimise-images`,
    "POST",
    {
      scale: zoom * (window.devicePixelRatio || 1),
    },
  );
  clearMedia();
  return result;
}

export async function optimiseImportedImages(
  results: ImportResult[],
  progress: (value: string) => void,
) {
  let optimised = 0;
  for (const result of results) {
    const pages = await api<Page[]>(`/workspaces/${result.id}/pages`);
    const start = pages.findIndex((page) => page.id === result.pageId);
    if (start < 0)
      throw new Error(
        "Imported pages are unavailable. Reopen the workspace to optimise them.",
      );
    for (const [index, page] of pages
      .slice(start, start + result.pageCount)
      .entries()) {
      progress(`Optimising page ${index + 1}/${result.pageCount}…`);
      optimised += (await optimisePageImages(page.id)).optimised;
    }
    refreshWorkspacePreview(result.id);
  }
  return optimised;
}
