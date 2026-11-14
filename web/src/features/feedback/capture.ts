/** Capture only the visible app, excluding dialogs and tooltip overlays. */
export async function captureView(): Promise<string> {
  const target = document.querySelector<HTMLElement>(".app");
  if (!target)
    throw new Error(
      "The current application view could not be captured. You can upload an image instead.",
    );
  const { default: html2canvas } = await import("html2canvas");
  const canvas = await html2canvas(target, {
    logging: false,
    useCORS: false,
    allowTaint: false,
    width: window.innerWidth,
    height: window.innerHeight,
    scale: Math.min(1, 1920 / window.innerWidth, 1080 / window.innerHeight),
    ignoreElements: (element) =>
      element.matches("dialog,[role=tooltip],.tooltip"),
  });
  return canvas.toDataURL("image/png").split(",")[1];
}
