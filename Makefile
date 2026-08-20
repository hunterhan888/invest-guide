.PHONY: dev backend-dev backend-test backend-build backend-vet backend-fmt frontend-dev frontend-build frontend-lint frontend-typecheck frontend-test test check

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

# 列出需格式化的文件；有输出则返回失败，作为门禁
backend-fmt:
	@cd backend && test -z "$$(gofmt -l .)" || (echo "以下文件未格式化:"; gofmt -l .; exit 1)

# 前端
frontend-dev:
	cd frontend && bun run dev

frontend-build:
	cd frontend && bun run build

frontend-lint:
	cd frontend && bun run lint

frontend-typecheck:
	cd frontend && bunx tsc --noEmit

frontend-test:
	cd frontend && bun run test

# 综合
test: backend-test frontend-test
	@echo "All tests passed"

dev: backend-dev

# 一键门禁：推送前必须通过
check: backend-fmt backend-vet backend-test frontend-lint frontend-typecheck frontend-test
	@echo "All checks passed"

# Docker
docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-build:
	docker compose build

docker-logs:
	docker compose logs -f
