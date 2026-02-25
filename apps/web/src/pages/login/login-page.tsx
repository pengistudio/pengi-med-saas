import GentooPenguin from "@/assets/gentoo.png";
import LoginImage from "@/assets/login.webp";
import SelectLanguage from "@/components/custom/select-language";
import LoginForm from "@/sections/forms/login/login-form";

const Login = () => {
	return (
		<>
			<div className="absolute top-6 left-6 flex items-center gap-1 font-bold">
				<img src={GentooPenguin} alt="Gentoo" className="w-8 h-8" />
				Gentoo
			</div>
			<main className="grid min-h-svh md:grid-cols-2 items-center justify-center gap-4">
				<div className="px-4 flex justify-center items-center w-full md:px-0 flex-col gap-4">
					<LoginForm />
					<SelectLanguage />
				</div>
				<div className="md:block hidden">
					<div className="relative">
						<div className="absolute w-full min-h-screen bg-primary-foreground opacity-90" />
						<img
							src={LoginImage}
							alt="Medical"
							className="w-full min-h-screen object-cover"
						/>
					</div>
				</div>
			</main>
		</>
	);
};

export default Login;
