import {
	authorizeXml,
	type ComprobanteModel,
	generateXmlInvoice,
	type SRIEnv,
	signXml,
	validateXml,
} from "osodreamer-sri-xml-signer";

export type SignParams = {
	p12Buffer: Uint8Array;
	xmlBuffer: Uint8Array;
	password: string;
};

export type ValidateParams = {
	xmlBuffer: Uint8Array;
	env: SRIEnv;
};

export type AuthorizeParams = {
	accessKey: string;
	env: SRIEnv;
};

export const signerService = {
	generate: (payload: ComprobanteModel) => generateXmlInvoice(payload),

	sign: ({ p12Buffer, xmlBuffer, password }: SignParams) =>
		signXml({ p12Buffer, xmlBuffer, password }),

	validate: ({ xmlBuffer, env }: ValidateParams) =>
		validateXml({ xml: xmlBuffer, env }),

	authorize: ({ accessKey, env }: AuthorizeParams) =>
		authorizeXml({ claveAcceso: accessKey, env }),
};
