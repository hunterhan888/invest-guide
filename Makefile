.PHONY: dev backend-dev backend-test backend-build backend-vet backend-fmt frontend-dev frontend-build test

# 后端
# backend-dev 依赖 .env（config.Load 自动从项目根向上加载），无需手动传环境变量
backend-dev:
	cd backend && go run ./cmd/server/

backend-test:
	cd backend && go test ./... -cover

backend-build:
	cd backend && go build -o bin/server ./cmd/server/

backend-mcp:
	cd backend && go build -o bin/mcp-server ./cmd/mcp-server/

backend-vet:
	cd backend && go vet ./...

backend-fmt:
	cd backend && gofmt -l .

# 前端
frontend-dev:
	cd frontend && bun run dev

frontend-build:
	cd frontend && bun run build

# 综合
test: backend-test
	@echo "All tests passed"

dev: backend-dev

# Docker
docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-build:
	docker compose build

docker-logs:
	docker compose logs -f
