import { ImagePreparationError, resizeImageToJpeg } from "@/common/utils/image";

const maxSizeMiB = 3;
const maxSize = maxSizeMiB * 1024 * 1024;
const coverFormats = { "image/jpeg": "JPEG", "image/png": "PNG" };

export const coverImageAccept = Object.keys(coverFormats).join(",");
export const coverImageFormatHint = `Выберите фото в формате ${Object.values(coverFormats).join(" или ")}.`;
export const coverImageSizeHint = `Выберите изображение размером до ${maxSizeMiB} МБ.`;

export async function prepareCoverImage(file: File) {
  if (!Object.hasOwn(coverFormats, file.type)) {
    throw new ImagePreparationError(coverImageFormatHint);
  }

  const blob = await resizeImageToJpeg(file, { maxEdge: 1920, quality: 0.85, background: "#fff" });
  if (blob.size > maxSize) throw new ImagePreparationError(coverImageSizeHint);

  return new File([blob], "cover.jpg", { type: "image/jpeg" });
}
