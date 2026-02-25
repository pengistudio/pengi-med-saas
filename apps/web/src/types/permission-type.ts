import type { BaseStringID } from "./api-type";

export type Permission = {
	Name: string;
	Category: string;
	Description: string;
} & BaseStringID;
