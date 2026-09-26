import {
	Select,
	SelectContent,
	SelectGroup,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@pengi/ui";
import { useLanguage } from "./language-context";
import { useMessageStore } from "./message-store";
import type { SupportedLocale } from "./zod-i18n";

export function SelectLanguage() {
	const { changeLanguage } = useLanguage();
	const { lang } = useMessageStore();
	return (
		<Select
			defaultValue={lang}
			onValueChange={(value: SupportedLocale | null) => {
				changeLanguage(value ?? "es");
			}}
		>
			<SelectTrigger className="w-fit">
				<SelectValue />
			</SelectTrigger>
			<SelectContent>
				<SelectGroup>
					<SelectItem value="es">Español</SelectItem>
					<SelectItem value="en">English</SelectItem>
				</SelectGroup>
			</SelectContent>
		</Select>
	);
}
