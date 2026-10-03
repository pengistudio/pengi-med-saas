const HEIC_TYPES = [
	"image/heic",
	"image/heif",
	"image/heic-sequence",
	"image/heif-sequence",
];
const HEIC_EXTENSION = /\.(heic|heif)$/i;

/**
 * Whether a file is HEIC/HEIF. iOS Safari often reports an empty type, so the
 * extension counts too.
 */
export function isHeicFile(file: File): boolean {
	return (
		HEIC_TYPES.includes(file.type.toLowerCase()) ||
		HEIC_EXTENSION.test(file.name)
	);
}

/**
 * HEIC/HEIF becomes a JPEG File (".jpg" name, image/jpeg); anything else is
 * returned untouched. The converter (a heavy WASM lib) is loaded only when a
 * HEIC file shows up, so it stays out of the main bundle.
 */
export async function convertHeicToJpeg(file: File): Promise<File> {
	if (!isHeicFile(file)) return file;
	const { heicTo } = await import("heic-to");
	const blob = await heicTo({ blob: file, type: "image/jpeg", quality: 0.9 });
	return new File([blob], `${file.name.replace(HEIC_EXTENSION, "")}.jpg`, {
		type: "image/jpeg",
		lastModified: file.lastModified,
	});
}
