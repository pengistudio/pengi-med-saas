import type { BaseModel, HttpService, ServiceResponse } from "../http";

export type MessageMap = Record<string, string>;

export interface UIMessage extends BaseModel {
	key: string;
	value: string;
	lang: string; // por ejemplo: 'es', 'en', etc.
}

let messagesService: HttpService | undefined;

/** Set by initShared: the (unauthenticated) client the messages come from. */
export function setMessagesService(service: HttpService) {
	messagesService = service;
}

export const getMessages = async (
	lang: string,
): Promise<ServiceResponse<UIMessage[]>> => {
	if (!messagesService) {
		throw new Error(
			"@pengi/shared: call initShared({ client }) before loading messages",
		);
	}
	return messagesService.get<UIMessage[]>(`/i18n/messages?lang=${lang}`);
};
