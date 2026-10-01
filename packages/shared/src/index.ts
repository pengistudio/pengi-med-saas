import type { AxiosInstance } from "axios";
import { setMessagesClient } from "./i18n/messages-service";

/**
 * Wires @pengi/shared to the app. Call once at startup, before rendering.
 * `client` is the app's unauthenticated axios instance: UI messages are
 * public.
 */
export function initShared({ client }: { client: AxiosInstance }) {
	setMessagesClient(client);
}

export {
	type BaseModel,
	type CustomAxiosRequestConfig,
	createHttpService,
	type ErrorResponse,
	HttpService,
	type ResponseError,
	type ServiceResponse,
	type SuccessResponse,
} from "./http";
export { AppTextBridge } from "./i18n/app-text-bridge";
export { parseDateOnly, toDateOnlyString } from "./i18n/date-only";
export { LanguageProvider, useLanguage } from "./i18n/language-context";
export { useMessageStore } from "./i18n/message-store";
export {
	getMessages,
	type MessageMap,
	type MessagesResult,
} from "./i18n/messages-service";
export { SelectLanguage } from "./i18n/select-language";
export { useMessages } from "./i18n/use-messages";
export {
	type AppText,
	type DateInput,
	type DateStyle,
	type TextValues,
	useText,
} from "./i18n/use-text";
export {
	type SupportedLocale,
	updateZodLocale,
	zodLocales,
} from "./i18n/zod-i18n";
