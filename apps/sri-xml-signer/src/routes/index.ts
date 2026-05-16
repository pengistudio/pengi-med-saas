import { Router } from "express";
import multer from "multer";
import { healthHandler } from "../handlers/health.handler";
import {
	authorizeHandler,
	generateHandler,
	signHandler,
	validateHandler,
} from "../handlers/signer.handler";
import { asyncHandler } from "../middleware/error.middleware";

const upload = multer({ storage: multer.memoryStorage() });

const router = Router();

router.get("/health", healthHandler);

router.post("/generate", asyncHandler(generateHandler));

router.post(
	"/sign",
	upload.fields([
		{ name: "p12", maxCount: 1 },
		{ name: "xml", maxCount: 1 },
	]),
	asyncHandler(signHandler),
);

router.post("/validate/:env", asyncHandler(validateHandler));
router.post("/authorization/:env", asyncHandler(authorizeHandler));

router.all("/{*splat}", (_req, res) => {
	res.status(404).json({
		status: "NOT_FOUND",
		message: "Ruta no encontrada",
		statusCode: 404,
	});
});

export default router;
