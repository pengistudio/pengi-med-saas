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

const LANGUAGES: Record<SupportedLocale, string> = {
	es: "Español",
	en: "English",
};

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
				<SelectValue>
					{(value: SupportedLocale) => LANGUAGES[value] ?? value}
				</SelectValue>
			</SelectTrigger>
			<SelectContent>
				<SelectGroup>
					{Object.entries(LANGUAGES).map(([value, label]) => (
						<SelectItem key={value} value={value}>
							{label}
						</SelectItem>
					))}
				</SelectGroup>
			</SelectContent>
		</Select>
	);
}
