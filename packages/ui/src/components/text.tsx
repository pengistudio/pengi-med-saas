import React from "react";
import { useUiText } from "../context/text-context";

type Props = {
	uuid?: string;
	/** Fills the message's `{name}` placeholders; `count` also picks the plural form. */
	values?: Record<string, string | number>;
	type?: "div" | "span" | "p" | "h1" | "h2" | "h3" | "h4" | "h5" | "h6";
	className?: string;
};

const Text = (props: Props) => {
	const { uuid = "empty", values, type = "span", className } = props;
	const { textGet } = useUiText();
	return React.createElement(type, { className }, textGet(uuid, values));
};

export { Text };
