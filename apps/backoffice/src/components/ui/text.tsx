import React from "react";
import { useText } from "@/hooks/use-text";

type Props = {
	uuid?: string;
	replace?: string[];
	type?: "div" | "span" | "p" | "h1" | "h2" | "h3" | "h4" | "h5" | "h6";
	className?: string;
};

const Text = (props: Props) => {
	const { uuid = "empty", replace = [], type = "span", className } = props;
	const { textGet } = useText();
	const parsedText = replaceTemplate(textGet(uuid), replace);
	return React.createElement(type, { className }, parsedText);

	function replaceTemplate(template: string, values: string[]): string {
		let index = 0;

		return template.replace(/#\{(.*?)\}/g, () => {
			return values[index++] ?? "";
		});
	}
};

export { Text };
