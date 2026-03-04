import type { SupportedLocale } from "@/config/zod-i18n";
import { useLanguage } from "@/contexts/language-context";
import { useMessageStore } from "@/store/message-store";
import {
	Select,
	SelectContent,
	SelectGroup,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "../ui/select";

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
			<SelectGroup>
				<SelectContent>
					<SelectItem value="es">Español</SelectItem>
					<SelectItem value="en">English</SelectItem>
				</SelectContent>
			</SelectGroup>
		</Select>
	);
};

export default SelectLanguage;
