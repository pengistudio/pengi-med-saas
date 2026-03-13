import type { Request, Response } from "express";
import type { SRIEnv } from "osodreamer-sri-xml-signer";
import { signerService } from "../services/signer.service";
import { toBuffer } from "../utils/buffer";
import { logger } from "../utils/logger";

export const generateHandler = async (req: Request, res: Response) => {
	const { generatedXml, invoiceJson } = await signerService.generate(req.body);
	res.json({
		xml: generatedXml,
		json: invoiceJson,
		accessKey: invoiceJson.factura.infoTributaria.claveAcceso,
	});
};

export const signHandler = async (req: Request, res: Response) => {
	const body = req.body as Record<string, unknown>;
	const { password } = body as { password: string };

	// Prefer multipart uploaded files; fall back to base64 fields in JSON body
	const files = (
		req as Request & { files?: Record<string, Express.Multer.File[]> }
	).files;

	const p12Buf = files?.p12?.[0]?.buffer ?? toBuffer(body.p12Buffer);
	const xmlBuf = files?.xml?.[0]?.buffer ?? toBuffer(body.xmlBuffer);

	if (p12Buf.length === 0 || xmlBuf.length === 0) {
		res.status(400).json({
			data: "Missing or invalid p12Buffer/xmlBuffer",
			error: "Missing p12 or xml data",
			message: "Datos faltantes",
			statusCode: 400,
		});
		return;
	}

	const p12ForSign = new Uint8Array(p12Buf);
	const xmlForSign = new Uint8Array(xmlBuf);

	logger.info("Signing XML", {
		p12Bytes: p12ForSign.length,
		xmlBytes: xmlForSign.length,
	});

	const signedXml = await signerService.sign({
		p12Buffer: p12ForSign,
		xmlBuffer: xmlForSign,
		password,
	});

	logger.info("Signed XML", { signedBytes: signedXml.length });

	res.json({
		data: { xml: signedXml },
		message: "Operación exitosa",
		statusCode: 200,
	});
};

export const validateHandler = async (req: Request, res: Response) => {
	const sriEnv: SRIEnv = req.params.env === "prod" ? "prod" : "test";
	const { xml } = req.body as {
		xml: Uint8Array | string | number[] | undefined;
	};

	if (!xml) {
		res
			.status(400)
			.json({ status: "INVALID", message: "XML requerido", statusCode: 400 });
		return;
	}

	const xmlBuffer =
		xml instanceof Uint8Array
			? xml
			: typeof xml === "string"
				? new Uint8Array(Buffer.from(xml, "base64"))
				: new Uint8Array(xml);

	logger.info("Validating XML", { xmlBytes: xmlBuffer.length, env: sriEnv });

	try {
		const result = await signerService.validate({ xmlBuffer, env: sriEnv });
		const estado = result?.estado ?? "UNKNOWN";
		const isValid = estado.toUpperCase() === "RECIBIDA";
		const statusCode = isValid ? 200 : 422;

		logger.info("Validation result", { estado, statusCode });

		res.status(statusCode).json({ status: estado, message: result?.mensaje, statusCode });
	} catch (err) {
		const message = err instanceof Error ? err.message : String(err);

		// SRI already has this key in processing — treat as received (proceed to authorization)
		if (message.toUpperCase().includes("EN PROCESAMIENTO")) {
			logger.info("SRI key already in processing, treating as RECIBIDA", { env: sriEnv });
			res.status(200).json({ status: "RECIBIDA", message, statusCode: 200 });
			return;
		}

		logger.warn("SRI validation error", { message, env: sriEnv });
		res.status(422).json({ status: "ERROR", message, statusCode: 422 });
	}
};

export const authorizeHandler = async (req: Request, res: Response) => {
	const sriEnv: SRIEnv = req.params.env === "prod" ? "prod" : "test";
	const { accessKey } = req.body as { accessKey?: string };

	if (!accessKey) {
		res.status(400).json({
			status: "INVALID",
			message: "Clave de acceso requerida",
			statusCode: 400,
		});
		return;
	}

	try {
		const authorization = await signerService.authorize({ accessKey, env: sriEnv });
		const estado =
			(authorization as unknown as Record<string, unknown>)?.estado ?? "UNKNOWN";
		const isAuthorized =
			typeof estado === "string" ? estado.toUpperCase() === "AUTORIZADO" : true;
		const statusCode = isAuthorized ? 200 : 422;

		res.status(statusCode).json({ ...authorization, status: estado, statusCode });
	} catch (err) {
		const message = err instanceof Error ? err.message : String(err);
		logger.warn("SRI authorization error", { message, accessKey, env: sriEnv });
		res.status(422).json({ status: "ERROR", message, statusCode: 422 });
	}
};
