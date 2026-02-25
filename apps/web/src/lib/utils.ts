import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

/**
 * Le da formato a la fecha según lo que se especifica
 * @param date - Fecha
 * @param options  - opciones en formato Intl.DateTimeFormatOptions (opcional)
 * @returns `string` con el formato de la fecha
 */
export function dateParser(
	date: Date | string,
	options?: Intl.DateTimeFormatOptions,
) {
	let transformDate = new Date();
	if (typeof date === "string") transformDate = new Date(date);
	else {
		transformDate = date;
	}
	if (options)
		return Intl.DateTimeFormat("es-EC", options).format(transformDate);
	return Intl.DateTimeFormat("es-EC", { dateStyle: "medium" }).format(
		transformDate,
	);
}
