export class ImagePreparationError extends Error {}

export async function resizeImageToJpeg(source: Blob, { maxEdge, quality, background }: {
  maxEdge: number;
  quality: number;
  background: string;
}) {
  const image = await createImageBitmap(source).catch(() => {
    throw new ImagePreparationError("Не удалось открыть изображение.");
  });

  try {
    const scale = Math.min(1, maxEdge / Math.max(image.width, image.height));
    const width = Math.max(1, Math.round(image.width * scale));
    const height = Math.max(1, Math.round(image.height * scale));
    const canvas = new OffscreenCanvas(width, height);
    const context = canvas.getContext("2d");
    if (!context) throw new ImagePreparationError("Не удалось подготовить изображение.");

    context.fillStyle = background;
    context.fillRect(0, 0, width, height);
    context.drawImage(image, 0, 0, width, height);

    return await canvas.convertToBlob({ type: "image/jpeg", quality });
  } finally {
    image.close();
  }
}
