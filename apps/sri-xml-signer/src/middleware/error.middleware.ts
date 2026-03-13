import type { NextFunction, Request, Response } from "express";
import { logger } from "../utils/logger";

export const asyncHandler =
	(fn: (req: Request, res: Response, next: NextFunction) => Promise<void>) =>
	(req: Request, res: Response, next: NextFunction) =>
		fn(req, res, next).catch(next);

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export const errorMiddleware = (
	err: Error,
	_req: Request,
	res: Response,
	_next: NextFunction,
) => {
	logger.error("Unhandled error", { message: err.message, stack: err.stack });
	res.status(500).json({
		status: "ERROR",
		message: "Internal server error",
		error: err.message,
		statusCode: 500,
	});
};
