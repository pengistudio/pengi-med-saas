import type {
	InvoicePaymentStatus,
	MovementCategory,
	MovementType,
	PaymentMethod,
} from "@/api/cash-service";

type TextGet = (key: string) => string;

// Literal keys (one per value) so the message-keys test checks every one.

export function paymentMethodLabel(textGet: TextGet, method: PaymentMethod) {
	switch (method) {
		case "cash":
			return textGet("cash.method.cash");
		case "transfer":
			return textGet("cash.method.transfer");
		case "credit_card":
			return textGet("cash.method.credit_card");
		case "debit_card":
			return textGet("cash.method.debit_card");
		case "check":
			return textGet("cash.method.check");
		default:
			return textGet("cash.method.other");
	}
}

export function movementTypeLabel(textGet: TextGet, type: MovementType) {
	return type === "in"
		? textGet("cash.movement.type.in")
		: textGet("cash.movement.type.out");
}

export function movementCategoryLabel(
	textGet: TextGet,
	category: MovementCategory,
) {
	switch (category) {
		case "expense":
			return textGet("cash.movement.category.expense");
		case "withdrawal":
			return textGet("cash.movement.category.withdrawal");
		case "deposit":
			return textGet("cash.movement.category.deposit");
		default:
			return textGet("cash.movement.category.adjustment");
	}
}

export function paymentStatusLabel(
	textGet: TextGet,
	status: InvoicePaymentStatus,
) {
	switch (status) {
		case "paid":
			return textGet("cash.payment_status.paid");
		case "partial":
			return textGet("cash.payment_status.partial");
		default:
			return textGet("cash.payment_status.unpaid");
	}
}
