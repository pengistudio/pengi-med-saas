import { noAuthApi } from ".";
import {
	type BaseModel,
	createHttpService,
	type ServiceResponse,
} from "./fetch";

const loginService = createHttpService(noAuthApi);

export type MessageMap = Record<string, string>;

export interface UIMessage extends BaseModel {
	key: string;
	value: string;
	lang: string; // por ejemplo: 'es', 'en', etc.
}

export const getMessages = async (
	lang: string,
): Promise<ServiceResponse<UIMessage[]>> => {
	return loginService.get<UIMessage[]>(`/i18n/messages?lang=${lang}`);
};
