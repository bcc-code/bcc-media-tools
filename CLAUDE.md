# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

BCC Media Tools is a fullstack application containing internal/backoffice tools for BCC Media:

- **Transcription Editor**: Word-level timestamped transcription editor with Mediabanken/BMM sync
- **BMM Uploader**: File upload with direct BMM integration via Temporal workflows
- **Shorts**: Submit short clips
- **Export / VB Export**: Export config, asset resolution, and timed-metadata export (incl. VideoBible)
- **Cantemo**: Trigger Cantemo actions
- **Vault**: Search and inspect Vault items (image, proxy, and waveform proxying)
- **Admin**: Permissions management

## Tech Stack

- **Frontend**: Nuxt 4 (Vue 3), TypeScript, TailwindCSS 4, in-house `Design*` component library
- **Backend**: Go 1.25+, ConnectRPC, Temporal SDK
- **API**: Protocol Buffers with ConnectRPC (gRPC-compatible over HTTP)
- **Auth**: Auth0 via `x-token-user-email` header

## Commands

### Development

```bash
# Start Temporal (required for backend)
temporal server start-dev

# Frontend (runs on port 8001)
cd frontend && pnpm dev

# Backend
cd backend && go run cmd/server/main.go

# Regenerate API code from protobuf
buf generate
```

### Backend

```bash
cd backend
go run cmd/server/main.go     # Run server
go test ./...                  # Run all tests
go test ./cmd/server/...       # Run server tests
```

### Frontend

```bash
cd frontend
pnpm dev        # Development server
pnpm build      # Production build
pnpm preview    # Preview production build
pnpm typecheck  # Type-check (vue-tsc)
pnpm format     # Format with Prettier
```

## Architecture

### API Layer (ConnectRPC)

API definitions live in `api/v1/api.proto`. Running `buf generate` creates:

- Backend: `backend/api/v1/api.pb.go` + `backend/api/v1/apiv1connect/`
- Frontend: `frontend/src/gen/api/v1/`

### Backend Structure

- `backend/cmd/server/` - Server entry point (`main.go`) and per-feature API handlers (`transcription.go`, `upload.go`, `bmm.go`, `shorts.go`, `export.go`, `cantemo.go`, `vault*.go`, `permissions.go`)
- `backend/api/v1/` - Generated protobuf code
- `backend/bmm/` - BMM-specific utilities (token, id)

### Frontend Structure

- `frontend/app/pages/` - Nuxt pages (transcription/, upload/, shorts/, vault/, export.vue, vb-export.vue, cantemo.vue, admin.vue)
- `frontend/app/components/` - Vue components organized by feature; shared UI primitives live in `components/design/` (`Design*`)
- `frontend/app/composables/` - Shared composables (e.g. `useTools`, `useNumberFormat`, `useCantemoActions`)
- `frontend/src/gen/` - Generated API client code
- `frontend/locales/` - i18n (en.json, nb.json)

## Code Style

- **Prettier**: tabWidth 4, with tailwindcss plugin
- **Go**: Standard Go formatting (gofmt), tabs
- **Frontend uses 4-space indentation** in TypeScript/Vue files
