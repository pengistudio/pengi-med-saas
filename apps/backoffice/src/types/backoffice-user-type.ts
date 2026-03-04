export interface Base {
	ID: number;
	createdAt: string;
	updatedAt: string;
	deletedAt?: string | null;
}

export interface BackofficeUser extends Base {
	name: string;
	user_name: string;
	refresh_token: string;
}

export interface LoginResponse {
	token: string;
	exchange_token: string;
	user_id: number;
}

export interface LoginRequest extends Record<string, unknown> {
	user_name: string;
	password: string;
}
