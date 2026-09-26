import type { SupportedLocale } from "@pengi/shared";
import { useLanguage, useMessageStore } from "@pengi/shared";
import {
	Select,
	SelectContent,
	SelectGroup,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@pengi/ui";

const SelectLanguage = () => {
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
};

export default SelectLanguage;
