import { useText } from "@pengi/shared";
import React from "react";
import { PasskeysCard } from "@/components/features/security/passkeys-card";
import { TwoFactorCard } from "@/components/features/security/two-factor-card";

/** The signed-in admin's own second factors: authenticator app and passkeys. */
const SecurityPage = () => {
	const { textGet } = useText();
	const [version, setVersion] = React.useState(0);

	return (
		<div className="mx-auto max-w-2xl space-y-6">
			<div className="space-y-1">
				<h1 className="text-2xl font-bold tracking-tight">
					{textGet("backoffice.security.title")}
				</h1>
				<p className="text-sm text-muted-foreground">
					{textGet("backoffice.security.description")}
				</p>
			</div>
			<TwoFactorCard version={version} />
			<PasskeysCard onChanged={() => setVersion((v) => v + 1)} />
		</div>
	);
};

export default SecurityPage;
