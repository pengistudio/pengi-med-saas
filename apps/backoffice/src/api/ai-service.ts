import { createHttpService, type ServiceResponse } from "@pengi/shared";
import { api } from ".";

const httpService = createHttpService(api);

export type AIFunction = "summary" | "scribe" | "bot";

/** Where the API key a function's calls use comes from. */
export type AIKeySource = "db" | "env" | "none";

/**
 * Provider and prices (USD per million tokens) of one AI function. Empty
 * fields use the server env default shown in `default_*`. The API key is
 * write-only: only `has_api_key` / `api_key_source` come back.
 */
export interface AIFunctionSettings {
	function: AIFunction;
	base_url: string;
	default_base_url: string;
	model: string;
	default_model: string;
	reasoning_effort: string;
	default_reasoning_effort: string;
	/** A key is stored in backoffice. */
	has_api_key: boolean;
	/** AI_API_KEY is set on the server. */
	env_has_api_key: boolean;
	api_key_source: AIKeySource;
	/** AI_ENCRYPTION_KEY is set: keys can be saved. */
	encryption_available: boolean;
	price_input: number;
	price_output: number;
}

/** What is saved for one function. Empty `api_key` keeps the stored one. */
export interface AISettingsUpdate {
	function: AIFunction;
	base_url: string;
	model: string;
	reasoning_effort: string;
	api_key: string;
	clear_api_key: boolean;
	price_input: number;
	price_output: number;
}

export interface AIUsageRow {
	tenant_id: number;
	tenant_name: string;
	function: AIFunction;
	uses: number;
	prompt_tokens: number;
	completion_tokens: number;
	cost_usd: number;
}

export interface AIUsageReport {
	month: string;
	rows: AIUsageRow[];
}

export const getAISettings = async (): Promise<
	ServiceResponse<AIFunctionSettings[]>
> =>
	httpService.get<AIFunctionSettings[]>("/backoffice/ai/settings", {
		notifyError: true,
	});

export const updateAISettings = async (
	settings: AISettingsUpdate[],
): Promise<ServiceResponse<AIFunctionSettings[]>> =>
	httpService.put<AIFunctionSettings[]>(
		"/backoffice/ai/settings",
		settings as unknown as Record<string, unknown>,
		{ notifySuccess: true, notifyError: true },
	);

/** `month` is YYYY-MM; empty is the current month. */
export const getAIUsage = async (
	month?: string,
): Promise<ServiceResponse<AIUsageReport>> =>
	httpService.get<AIUsageReport>("/backoffice/ai/usage", {
		params: month ? { month } : undefined,
		notifyError: true,
	});
