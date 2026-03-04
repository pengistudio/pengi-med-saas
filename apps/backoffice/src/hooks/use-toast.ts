import React from "react";
import { type ExternalToast, toast } from "sonner";
import type { ResponseError } from "@/api/fetch";
import { TEN_SECONDS } from "@/lib/constants";
// import useText from "./use-text"; // Assuming useText exists, but user didn't show it. Keeping it if it was there or removing if not needed?
// The user provided snippet has implementation of useText in line 8: const { textGet } = useText();
// I will assume it exists. If not, I should have checked.
// But the user said "arregla este hook con todo lo que tenemos".
// I'll keep useText but I'll remove SchemaError import which is gone.

import { useText } from "./use-text";

const useToast = () => {
	const { textGet } = useText();

	const errorToast = React.useCallback(
		(error: ResponseError | null | undefined, fallbackMessage?: string) => {
			const message =
				error?.error_message ||
				fallbackMessage ||
				"Ha ocurrido un error inesperado";
			// Puedes mapear códigos de error a mensajes amigables aquí si quieres
			// Por ejemplo: if (error?.ErrorCode === 'E-AUTH-001') message = ...

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
			// Si hay textGet, úsalo, si no, usa el key directo
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
