# Run the API in development mode with hot-reload
dev:
	docker compose -f docker-compose.dev.yaml up --build

# Check formatting and lint for all TS/JS apps
check:
	npx @biomejs/biome check ./apps/web ./apps/backoffice


lint:
	npx @biomejs/biome check --write ./apps/web ./apps/backoffice

# Configure git to use shared hooks (run once after cloning)
setup:
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-commit
	@echo "Git hooks configured."

# Run all tests (backend + frontend)
tests:
	@echo "Running backend tests..."
	docker compose -f docker-compose.dev.yaml exec api go test ./... -v
	@echo "Running frontend tests..."
	docker compose -f docker-compose.dev.yaml exec web pnpm test:run

# Run backend tests only
tests-api:
	docker compose -f docker-compose.dev.yaml exec api go test ./... -v

# Run frontend tests only
tests-web:
	docker compose -f docker-compose.dev.yaml exec web pnpm test:run

# Run E2E tests (requires running stack with `just dev`)
tests-e2e:
	docker compose -f docker-compose.dev.yaml exec web pnpm exec playwright test

# Run E2E tests with UI browser visible
tests-e2e-ui:
	docker compose -f docker-compose.dev.yaml exec web pnpm exec playwright test --ui

# Run E2E tests in debug mode
tests-e2e-debug:
	docker compose -f docker-compose.dev.yaml exec web pnpm exec playwright test --debug