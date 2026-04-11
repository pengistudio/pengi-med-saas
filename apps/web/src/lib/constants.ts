export const EMPTY_STRING = "";
export const EMPTY_SPACE_STRING = " ";
export const ONE_SECOND = 1000;

export const TIMEOUT_API = 15 * ONE_SECOND;

export const TEN_SECONDS = 10000;

//HTTP CODES
export const STATUS_CODES = {
	REQUEST_TIMEOUT: 408,
	OK: 200,
	INTERNAL_SERVER_ERROR: 500,
	STATUS_INVALID_CREDENTIALS: 401,
};

export const ZERO = 0;
export const ONE = 1;
export const FIVE = 5;

export const BACK_ROUTE = -1;

// Tax Codes
export type TaxCode = "2" | "3" | "5"; // 2 = IVA, 3 = ICE, 5 = IRBPNR

export const TAX_CODES: Record<TaxCode, string> = {
	"2": "IVA",
	"3": "ICE",
	"5": "IRBPNR",
};

// Tax Percentage Codes
export type TaxPercentageCode =
	| "0" // 0%
	| "2" // 12%
	| "3" // 14%
	| "4" // 15%
	| "5" // 5%
	| "6" // No objeto de IVA
	| "7" // Exento de IVA
	| "8" // IVA diferenciado
	| "10"; // 13% (temporal)

export const IVA_PERCENTAGE_CODES: Record<TaxPercentageCode, string> = {
	"0": "0% IVA",
	"2": "12% IVA",
	"3": "14% IVA (histórico)",
	"4": "15% IVA",
	"5": "5% IVA",
	"6": "No objeto de IVA",
	"7": "Exento de IVA",
	"8": "IVA diferenciado",
	"10": "13% IVA",
};

export const IVA_PERCENTAGE_CODES_AS_NUMBER: Record<TaxPercentageCode, number> =
	{
		"0": 0,
		"2": 0.12,
		"3": 0.14,
		"4": 0.15,
		"5": 0.05,
		"6": 0,
		"7": 0,
		"8": 0,
		"10": 0.13,
	};

export const PAYMENT_LABELS: Record<string, string> = {
	"01": "Sin utilización del sistema financiero",
	"15": "Compensación de deudas",
	"16": "Tarjeta prepago",
	"17": "Tarjeta de débito",
	"18": "Tarjeta de crédito",
	"19": "Dinero electrónico",
	"20": "Otros con utilización del sistema financiero",
	"21": "Endoso de títulos",
};

export const PERMISSIONS = {
	MEDICAL_RECORD: {
		PERMISSION_READ_PATIENT: "READ_PATIENT",
		PERMISSION_CREATE_PATIENT: "CREATE_PATIENT",
		PERMISSION_UPDATE_PATIENT: "UPDATE_PATIENT",
		PERMISSION_DELETE_PATIENT: "DELETE_PATIENT",
		PERMISSION_READ_MEDICAL_RECORD: "READ_MEDICAL_RECORD",
		PERMISSION_CREATE_MEDICAL_RECORD: "CREATE_MEDICAL_RECORD",
		PERMISSION_UPDATE_MEDICAL_RECORD: "UPDATE_MEDICAL_RECORD",
		PERMISSION_UPDATE_PRESCRIPTION: "UPDATE_PRESCRIPTION",
		PERMISSION_DOWNLOAD_PATIENT_REPORT: "DOWNLOAD_PATIENT_REPORT",
	},
	BILLING: {
		PERMISSION_READ_BILLING: "READ_BILLING",
		PERMISSION_CREATE_BILLING: "CREATE_BILLING",
		PERMISSION_UPDATE_BILLING: "UPDATE_BILLING",
		PERMISSION_DELETE_BILLING: "DELETE_BILLING",
		PERMISSION_MANAGE_SRI_SETTINGS: "MANAGE_SRI_SETTINGS",
	},
	TEAM: {
		PERMISSION_READ_TEAM: "READ_TEAM",
		PERMISSION_CREATE_TEAM: "CREATE_TEAM",
		PERMISSION_UPDATE_TEAM: "UPDATE_TEAM",
		PERMISSION_DELETE_TEAM: "DELETE_TEAM",
		PERMISSION_MANAGE_TEAM_MEMBERS: "MANAGE_TEAM_MEMBERS",
	},
	KANBAN: {
		PERMISSION_READ_KANBAN: "READ_KANBAN",
		PERMISSION_CREATE_KANBAN: "CREATE_KANBAN",
		PERMISSION_UPDATE_KANBAN: "UPDATE_KANBAN",
		PERMISSION_DELETE_KANBAN: "DELETE_KANBAN",
	},
};
