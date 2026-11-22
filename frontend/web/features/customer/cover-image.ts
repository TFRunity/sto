const maxSize = 3 * 1024 * 1024;
const maxEdge = 1920;

export class CoverImageError extends Error {}

export async function prepareCoverImage(file: File) {
  if (file.type !== "image/jpeg" && file.type !== "image/png") {
    throw new CoverImageError("Выберите изображение JPEG или PNG.");
  }

  const image = await createImageBitmap(file).catch(() => {
    throw new CoverImageError("Не удалось открыть изображение.");
  });

  try {
    const scale = Math.min(1, maxEdge / Math.max(image.width, image.height));
    const width = Math.max(1, Math.round(image.width * scale));
    const height = Math.max(1, Math.round(image.height * scale));
    const canvas = new OffscreenCanvas(width, height);
    const context = canvas.getContext("2d");
    if (!context) throw new CoverImageError("Не удалось подготовить изображение.");

    // Для прозрачных PNG оставляем белый фон при переводе в JPEG.
    context.fillStyle = "#fff";
    context.fillRect(0, 0, width, height);
    context.drawImage(image, 0, 0, width, height);

    const blob = await canvas.convertToBlob({ type: "image/jpeg", quality: 0.85 });
    if (blob.size > maxSize) throw new CoverImageError("Выберите изображение размером до 3 МБ.");

    return new File([blob], "cover.jpg", { type: "image/jpeg" });
  } finally {
    image.close();
  }
}
