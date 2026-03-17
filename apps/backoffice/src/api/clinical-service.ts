import type { BaseModel } from "./fetch";

export interface Patient extends BaseModel {
	tenant_id: number;
	document: string;
	phone: string;
	email: string;
	first_name: string;
	last_name: string;
	full_name?: string;
	birth_date: string;
	institution: string;
	gender: string;
	notes: string;
	insurance: string;
	medic: string;
	diagnosis: string;
	critical: boolean;
	app: string;
	apf: string;
	apqx: string;
	allergies?: string;
}
