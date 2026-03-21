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