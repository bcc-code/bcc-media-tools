api: backend/api/v1/api.pb.go frontend/src/gen/api/v1/api_pb.ts

backend/api/v1/api.pb.go: api/v1/api.proto
	buf generate

.PHONY: run temporal backend frontend

# Load nvm's default node onto PATH when nvm is installed (so pnpm is found),
# otherwise rely on the ambient PATH.
NVM_SH := $(HOME)/.nvm/nvm.sh
USE_NODE := if [ -s "$(NVM_SH)" ]; then . "$(NVM_SH)" >/dev/null 2>&1; nvm use >/dev/null 2>&1 || nvm use node >/dev/null 2>&1; fi;

# Run the full local dev stack (Temporal + backend + frontend). Ctrl-C stops all.
run:
	@trap 'kill 0' EXIT INT TERM; \
	temporal server start-dev & \
	$(MAKE) backend & \
	$(MAKE) frontend & \
	wait

# Local Temporal dev server (required by the backend)
temporal:
	temporal server start-dev

# Backend API server (http://localhost:8080)
backend:
	cd backend && go run ./cmd/server

# Frontend dev server (http://localhost:8001)
frontend:
	cd frontend && $(USE_NODE) pnpm dev
