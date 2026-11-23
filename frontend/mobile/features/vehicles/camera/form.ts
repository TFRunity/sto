import { File } from "expo-file-system";
import { ImageManipulator, SaveFormat } from "expo-image-manipulator";

const maxEdge = 1920;

export function discardInspectionPhoto(uri: string) {
  try {
    const file = new File(uri);
    if (file.exists) file.delete();
  } catch {
    // Android can clear the cache while the camera screen is open.
  }
}

export async function prepareInspectionPhoto({ uri, width, height }: {
  uri: string;
  width: number;
  height: number;
}) {
  const context = ImageManipulator.manipulate(uri);
  try {
    if (Math.max(width, height) > maxEdge) {
      context.resize(width >= height ? { width: maxEdge } : { height: maxEdge });
    }

    const image = await context.renderAsync();
    try {
      const photo = await image.saveAsync({ format: SaveFormat.JPEG, compress: 0.75 });
      return photo.uri;
    } finally {
      image.release();
    }
  } finally {
    context.release();
  }
}
