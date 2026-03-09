import cors from "cors";
import dotenv from "dotenv";
import express, { type Request, type Response } from "express";
import * as fs from "fs";
import multer from "multer";
import * as os from "os";
import {
	authorizeXml,
	type ComprobanteModel,
	generateXmlInvoice,
	type SRIEnv,
	signXml,
	validateXml,
} from "osodreamer-sri-xml-signer";
import * as path from "path";

dotenv.config();

const app = express();
const port = 9000;

app.use(cors());
// Increase JSON/body size limit to accept base64-encoded files
app.use(express.json({ limit: "10mb" }));
app.use(express.urlencoded({ extended: true, limit: "10mb" }));

// Multer in-memory storage for multipart/form-data uploads
const upload = multer({ storage: multer.memoryStorage() });

app.get("/health", (_req: Request, res: Response) => {
	res.json({ status: "ok", service: "sri-xml-signer-service" });
});

app.post("/generate", async (req: Request, res: Response) => {
	const payload: ComprobanteModel = req.body;
	const { generatedXml, invoiceJson } = await generateXmlInvoice(payload);
	res.json({
		xml: generatedXml,
		json: invoiceJson,
		accessKey: invoiceJson.factura.infoTributaria.claveAcceso,
	});
});

// Support both JSON (base64) and multipart/form-data file uploads.
app.post(
	"/sign",
	// multer middleware will only parse multipart requests; it will be ignored for JSON
	upload.fields([
		{ name: "p12", maxCount: 1 },
		{ name: "xml", maxCount: 1 },
	]),
	async (req: Request, res: Response) => {
		const body = req.body as any;
		const { password } = body;

		// Helper to normalize incoming payloads into Buffer
		const toBuffer = (input: any): Buffer => {
			if (!input) return Buffer.alloc(0);
			if (typeof input === "string") return Buffer.from(input, "base64"); // base64 -> Buffer
			if (Buffer.isBuffer(input)) return input;
			if (input instanceof Uint8Array) return Buffer.from(input);
			if (Array.isArray(input))
				return Buffer.from(new Uint8Array(input as number[]));
			// fallback
			try {
				return Buffer.from(input);
			} catch {
				return Buffer.alloc(0);
			}
		};

		// If multer parsed files, prefer those (they are in memory buffers)
		const files = (req as any).files as
			| Record<string, Express.Multer.File[]>
			| undefined;
		let p12Buf: Buffer;
		let xmlBuf: Buffer;

		if (files && (files["p12"] || files["xml"])) {
			const p12File = files["p12"] && files["p12"][0];
			const xmlFile = files["xml"] && files["xml"][0];
			p12Buf = p12File ? p12File.buffer : toBuffer(body.p12Buffer);
			xmlBuf = xmlFile ? xmlFile.buffer : toBuffer(body.xmlBuffer);
		} else {
			p12Buf = toBuffer(body.p12Buffer);
			xmlBuf = toBuffer(body.xmlBuffer);
		}

		// If any buffer is empty -> error
		if (p12Buf.length === 0 || xmlBuf.length === 0) {
			return res.status(400).json({
				data: "Missing or invalid p12Buffer/xmlBuffer",
				error: "Missing p12 or xml data",
				message: "Datos faltantes",
				statusCode: 400,
			});
		}

		// Create unique temp dir and write files for libraries that expect files on disk
		const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "sri-"));
		const p12Path = path.join(tmpDir, "cert.p12");
		const xmlPath = path.join(tmpDir, "invoice.xml");

		try {
			// write files; set restrictive perms for p12
			fs.writeFileSync(p12Path, p12Buf, { mode: 0o600 });
			fs.writeFileSync(xmlPath, xmlBuf);

			// read back and convert to Uint8Array for signXml
			const p12ForSign = new Uint8Array(fs.readFileSync(p12Path));
			const xmlForSign = new Uint8Array(fs.readFileSync(xmlPath));
			// Diagnostic: log sizes (avoid logging content)
			console.log(
				"Signing: p12 bytes=",
				p12ForSign.length,
				"xml bytes=",
				xmlForSign.length,
			);

			const signedXml = await signXml({
				p12Buffer: p12ForSign,
				password,
				xmlBuffer: xmlForSign,
			});

			console.log("Signed XML size:", signedXml.length);
			res.json({
				data: { xml: signedXml },
				message: "Operación exitosa",
				statusCode: 200,
			});
		} catch (error) {
			res.status(500).json({
				message: "Failed to sign XML",
				data: String(error),
				error: "Signing error",
				statusCode: 500,
			});
		} finally {
			// cleanup
			try {
				if (fs.existsSync(p12Path)) fs.unlinkSync(p12Path);
				if (fs.existsSync(xmlPath)) fs.unlinkSync(xmlPath);
				if (fs.existsSync(tmpDir)) fs.rmdirSync(tmpDir);
			} catch (cleanupErr) {
				// no-op (avoid masking original error)
				console.error("Cleanup error:", cleanupErr);
			}
		}
	},
);

app.post("/validate/:env", async (req: Request, res: Response) => {
	const { env } = req.params;
	const { xml }: { xml: Uint8Array | string | number[] } = req.body;

	const sriEnv: SRIEnv = env === "prod" ? "prod" : "test";

	if (!xml) {
		return res
			.status(400)
			.json({ status: "INVALID", message: "XML requerido", statusCode: 400 });
	}

	// Normaliza a Uint8Array para la librería
	const xmlBuffer =
		xml instanceof Uint8Array
			? xml
			: typeof xml === "string"
				? new Uint8Array(Buffer.from(xml, "base64"))
				: new Uint8Array(xml);

	console.log("Validating XML of size:", xmlBuffer.length);

	try {
		const validate = await validateXml({ xml: xmlBuffer, env: sriEnv });
		const estado = validate?.estado ?? "UNKNOWN";
		// SRI devuelve "RECIBIDA" cuando pasa validación; cualquier otro estado lo marcamos como error de negocio
		const isValid = estado.toUpperCase() === "RECIBIDA";
		const statusCode = isValid ? 200 : 422;

		console.log("Validation result:", validate);

		return res
			.status(statusCode)
			.json({ status: estado, message: validate?.mensaje, statusCode });
	} catch (error) {
		console.error("Validate error:", error);
		return res.status(500).json({
			status: "ERROR",
			message: "Error validando XML",
			error: String(error),
			statusCode: 500,
		});
	}
});

app.post("/authorization/:env", async (req: Request, res: Response) => {
	const { env } = req.params;
	const { accessKey }: { accessKey: string } = req.body;
	const sriEnv: SRIEnv = env === "prod" ? "prod" : "test";

	if (!accessKey) {
		return res.status(400).json({
			status: "INVALID",
			message: "Clave de acceso requerida",
			statusCode: 400,
		});
	}

	try {
		const authorization = await authorizeXml({
			claveAcceso: accessKey,
			env: sriEnv,
		});

		const estado = (authorization as any)?.estado ?? "UNKNOWN";
		const isAuthorized =
			typeof estado === "string" ? estado.toUpperCase() === "AUTORIZADO" : true;
		const statusCode = isAuthorized ? 200 : 422;

		return res
			.status(statusCode)
			.json({ ...authorization, status: estado, statusCode });
	} catch (error) {
		console.error("Authorization error:", error);
		return res.status(500).json({
			status: "ERROR",
			message: "Error autorizando factura",
			error: String(error),
			statusCode: 500,
		});
	}
});

// Fallback for undefined routes
app.all("*", (_req: Request, res: Response) => {
	return res.status(404).json({
		status: "NOT_FOUND",
		message: "Ruta no encontrada",
		statusCode: 404,
	});
});

app.listen(port, () => {
	console.log(`[server]: Server is running at http://localhost:${port}`);
});
