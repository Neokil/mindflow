import { readGoodnotes } from "./read";
import { writeGoodnotes } from "./write";
import type { Notebook } from "./archive";
self.onmessage = async (
  event: MessageEvent<{
    mode: "read" | "write";
    data: Uint8Array;
    notebook?: Notebook;
  }>,
) => {
  try {
    if (event.data.mode === "read") {
      const result = readGoodnotes(event.data.data);
      self.postMessage(
        { result },
        {
          transfer: [
            ...new Set(
              result.assets.map((asset) => asset.data.buffer as ArrayBuffer),
            ),
          ],
        },
      );
    } else
      self.postMessage({ result: await writeGoodnotes(event.data.notebook!) });
  } catch (error) {
    self.postMessage({
      error:
        (error as Error).message ||
        "The Goodnotes document could not be converted.",
    });
  }
};
