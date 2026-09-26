import React from "react";
import { useMessageStore } from "./message-store";

const useText = () => {
	const { messages } = useMessageStore();

	const textGet = React.useCallback(
		(key: string) => {
			const message = messages[key];
			if (!message) return `*${key}*`;
			return message;
		},
		[messages],
	);

	return { textGet };
};

export { useText };
