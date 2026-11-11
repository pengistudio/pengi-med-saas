# Reports API contract (as built)
GET /reports/{revenue|receivables|appointments|clinical}?from&to&doctor_id&format=csv|pdf (default last 30d, max 366d). Errors E-REPORTS-001 (filters), -002 (range), -003 (format). Shapes: apps/api/features/reports/dto/reports-dto.go.
