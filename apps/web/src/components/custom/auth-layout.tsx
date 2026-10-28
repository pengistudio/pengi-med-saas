import { SelectLanguage, useText } from "@pengi/shared";
import GentooPenguin from "@/assets/gentoo.png";
import LoginImage from "@/assets/login.webp";
import { LANDING_URL, SUPPORT } from "@/lib/constants";
import { generateWhatsAppLink } from "@/lib/utils";

/**
 * Shell of the public auth pages (login, signup, reset password): brand and
 * language on top, the form in the middle, legal and help links at the
 * bottom, and the penguin photo on the right from md up.
 */
const AuthLayout = ({ children }: { children: React.ReactNode }) => {
	const { textGet } = useText();
	const footerLinks = [
		{ href: `${LANDING_URL}/terms-of-service`, label: "auth.footer.terms" },
		{ href: `${LANDING_URL}/privacy-policy`, label: "auth.footer.privacy" },
		{
			href: generateWhatsAppLink(
				SUPPORT.WHATSAPP,
				textGet("auth.footer.help.message"),
			),
			label: "auth.footer.help",
		},
	];

	return (
		<main className="grid md:grid-cols-2 h-svh">
			<div className="overflow-y-auto flex flex-col">
				<header className="flex items-center justify-between gap-4 p-4 md:p-6">
					<div className="flex items-center gap-1.5 font-bold">
						<img src={GentooPenguin} alt="" className="w-8 h-8" />
						Gentoo
					</div>
					<SelectLanguage />
				</header>
				<div className="flex-1 flex justify-center items-center px-4 py-8">
					<div className="w-full max-w-md">{children}</div>
				</div>
				<footer className="flex flex-wrap justify-center gap-x-6 gap-y-2 p-6 text-sm text-muted-foreground">
					{footerLinks.map((link) => (
						<a
							key={link.label}
							href={link.href}
							target="_blank"
							rel="noreferrer"
							className="hover:text-foreground hover:underline underline-offset-4"
						>
							{textGet(link.label)}
						</a>
					))}
				</footer>
			</div>
			<div className="md:block hidden sticky top-0 h-svh">
				<div className="relative h-full overflow-hidden bg-primary">
					<img
						src={LoginImage}
						alt=""
						className="w-full h-full object-cover grayscale mix-blend-multiply"
					/>
					<div className="absolute inset-0 bg-linear-to-t from-black/60 via-black/10 to-transparent" />
					<p className="absolute bottom-0 left-0 p-10 max-w-md text-4xl font-semibold leading-tight tracking-tight text-white text-balance">
						{textGet("auth.tagline")}
					</p>
				</div>
			</div>
		</main>
	);
};

export default AuthLayout;
