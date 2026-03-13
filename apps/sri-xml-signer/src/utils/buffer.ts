/**
 * Normalizes any incoming payload shape into a Buffer.
 * Accepts: base64 string, Buffer, Uint8Array, or number[].
 */
export const toBuffer = (input: unknown): Buffer => {
	if (!input) return Buffer.alloc(0);
	if (typeof input === "string") return Buffer.from(input, "base64");
	if (Buffer.isBuffer(input)) return input;
	if (input instanceof Uint8Array) return Buffer.from(input);
	if (Array.isArray(input))
		return Buffer.from(new Uint8Array(input as number[]));
	try {
		return Buffer.from(input as ArrayBufferLike);
	} catch {
		return Buffer.alloc(0);
	}
};
