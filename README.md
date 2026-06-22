# BCC Media Tools

Contains several tools used for various internal/backoffice tasks.

Current tools:

- BMM Upload
- Transcription Editor
- Export
- VB Export
- Shorts generation
- Vault
- Admin

Access to each tool is controlled per-user via permissions (see `permissions.json`).

## Tools

### BMM Upload

Allows uploading localized audio files, with a direct link to BMM.
The files are ingested using a Temporal workflow after upload.

The main functionality is the ability to select a track from BMM and upload a file to it.

### Transcription Editor

Allows editing word level timestamped transcriptions of video files in JSON format,
and downloading them.
It is possible to synchronize the transcription with the preview from Mediabanken.

### Export

Trigger export workflows for media items. Supports mass-export to configured
destinations as well as timed metadata.

### VB Export

Trigger VB (playout) export workflows to configured destinations.

### Shorts generation

Generate shorts (short-form vertical videos) from existing videos.

### Vault

Search Mediabanken and preview items.

### Admin

Manage users and permissions. Only available to admins.

## Development

### Requirements

- Node.js
- pnpm
- Golang
- Temporal dev server ([Instructions](https://learn.temporal.io/getting_started/go/dev_environment/#set-up-a-local-temporal-service-for-development-with-temporal-cli))
- Docker (if you want to build images)
- ConnectRPC tools (see below)

### Setup

Install connectrpc tools:

```bash
# GO tools
go install github.com/bufbuild/buf/cmd/buf@latest
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest

# TypeScript/JS tools
# This are installed globally, you can also install them locally if you want to mess around
npm install -g @connectrpc/protoc-gen-connect-es @bufbuild/protoc-gen-es
```

If you get issues where protobuf cannot find the go plugins, ensure your Go bin directory is in your PATH.

```sh
export PATH=$PATH:$(go env GOPATH)/bin
```

Install dependencies:

```bash
cd frontend
pnpm install

cd ../backend
go mod download

cd ..
```

Update Config:

```bash
cp frontend/env.example frontend/.env
vim frontend/.env

cp backend/env.example backend/.env
vim backend/.env
```

Update intial permissions:

```bash
cp permissions.example.json permissions.json
vim permissions.json
```

### Development

Start temporal:

```bash
temporal server start-dev
```

Start the frontend:

```bash
cd frontend && pnpm dev
```

Start the backend:

```bash
cd backend && go run cmd/server/main.go
```

Regenerate api code:

```bash
buf generate
```

#### Working with the API

The api is defined in `api/v1/api.proto` (with shared types in `api/v1/common.proto`).
The definitions are written in protobuf, and the code is generated using `buf` and `protoc-gen-connect-go`.

More information can be found at:

- [ConnectRPC](https://connectrpc.com/)
- [Protobuf 3](https://protobuf.dev/programming-guides/proto3/)
- [gRPC](https://grpc.io/docs/languages/go/basics/)
