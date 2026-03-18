import GentooPenguin from "@/assets/gentoo.png";
import LoginImage from "@/assets/login.webp";
import SelectLanguage from "@/components/custom/select-language";
import SignupForm from "@/sections/forms/signup/signup-form";

const Signup = () => {
	return (
		<main className="grid md:grid-cols-2 h-svh">
			<div className="overflow-y-auto flex flex-col px-4 md:px-0">
				<div className="flex items-center gap-1 font-bold p-6">
					<img src={GentooPenguin} alt="Gentoo" className="w-8 h-8" />
					Gentoo
				</div>
				<div className="flex-1 flex justify-center items-center py-6">
					<div className="w-full flex flex-col items-center gap-4">
						<SignupForm />
						<SelectLanguage />
					</div>
				</div>
			</div>
			<div className="md:block hidden sticky top-0 h-svh">
				<div className="relative h-full">
					<div className="absolute inset-0 bg-primary-foreground opacity-90" />
					<img
						src={LoginImage}
						alt="Medical"
						className="w-full h-full object-cover"
					/>
				</div>
			</div>
		</main>
	);
};

export default Signup;
