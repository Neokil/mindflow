const maxBytes = 5 * 1024 * 1024;
/** Decode browser-supported images and send only a bounded, static PNG to the server. */
export async function imageAttachment(file: File): Promise<string> {
  if (file.size > maxBytes || (file.type && !file.type.startsWith("image/")))
    throw new Error("Choose an image up to 5 MB.");
  const url = URL.createObjectURL(file);
  try {
    const image = new Image();
    image.src = url;
    try {
      await image.decode();
    } catch {
      throw new Error(
        "This image format could not be read. Try PNG, JPEG, WebP, GIF, AVIF, or BMP.",
      );
    }
    const width = image.naturalWidth,
      height = image.naturalHeight;
    if (
      !width ||
      !height ||
      width > 8192 ||
      height > 8192 ||
      width * height > 16_000_000
    )
      throw new Error(
        "Choose an image up to 16 million pixels and 8,192 pixels per side.",
      );
    const canvas = document.createElement("canvas");
    canvas.width = width;
    canvas.height = height;
    const context = canvas.getContext("2d");
    if (!context) throw new Error("The image could not be prepared.");
    context.drawImage(image, 0, 0);
    const encoded = canvas.toDataURL("image/png").split(",")[1];
    if (encoded.length > Math.ceil(maxBytes / 3) * 4)
      throw new Error(
        "The image is too large after conversion. Choose a smaller image.",
      );
    return encoded;
  } finally {
    URL.revokeObjectURL(url);
  }
}
