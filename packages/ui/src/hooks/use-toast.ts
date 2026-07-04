import React from "react";
import { type ExternalToast, toast } from "sonner";
import { useUiText } from "../context/text-context";

const TEN_SECONDS = 10000;

export type ResponseError = {
	error_code: string;
	error_message: string;
};

const useToast = () => {
	const { textGet } = useUiText();

	const errorToast = React.useCallback(
		(error: ResponseError | null | undefined, fallbackMessage?: string) => {
			const message =
				error?.error_message ||
				fallbackMessage ||
				"Ha ocurrido un error inesperado";

			toast.error(message, {
				description: error?.error_code
					? `Código: ${error.error_code}`
					: undefined,
				duration: TEN_SECONDS,
			});
		},
		[],
	);

	const successToast = React.useCallback(
		(messageKey?: string) => {
			const message = textGet
				? textGet(messageKey || "toast.success")
				: messageKey;
			toast.success(message, {
				duration: TEN_SECONDS,
			});
		},
		[textGet],
	);

	const infoToast = React.useCallback(
		(title: string | React.ReactNode, data?: ExternalToast) => {
			toast.info(title, {
				...data,
				duration: TEN_SECONDS,
			});
		},
		[],
	);

	return { errorToast, infoToast, successToast };
};

export default useToast;
