# Run the API in development mode with hot-reload
dev:
	docker compose -f docker-compose.dev.yaml up --build

# Check formatting and lint for all TS/JS apps
check:
	npx @biomejs/biome ci ./apps/web ./apps/backoffice
