import cors from "cors";
import express from "express";
import helmet from "helmet";
import morgan from "morgan";
import { config } from "./config";
import { errorMiddleware } from "./middleware/error.middleware";
import router from "./routes";

const app = express();

app.use(helmet());
app.use(cors({ origin: config.allowedOrigin }));
app.use(morgan("combined"));
app.use(express.json({ limit: "10mb" }));
app.use(express.urlencoded({ extended: true, limit: "10mb" }));

app.use(router);
app.use(errorMiddleware);

export default app;
