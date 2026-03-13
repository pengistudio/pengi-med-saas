import dotenv from "dotenv";

dotenv.config();

export const config = {
	port: Number(process.env.PORT) || 9000,
	allowedOrigin: process.env.ALLOWED_ORIGIN || "http://localhost:8080",
};
