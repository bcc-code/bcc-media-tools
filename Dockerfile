FROM node:20-slim AS base
ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
RUN corepack enable && npm install -g corepack@latest
COPY ./frontend /app
WORKDIR /app

FROM base AS deps
RUN --mount=type=cache,id=pnpm,target=/pnpm/store pnpm install --frozen-lockfile

FROM base AS build
RUN --mount=type=cache,id=pnpm,target=/pnpm/store pnpm install --frozen-lockfile
ARG NUXT_PUBLIC_GRPC_URL
ENV NUXT_PUBLIC_GRPC_URL=$NUXT_PUBLIC_GRPC_URL
ARG NUXT_PUBLIC_RUDDERSTACK_WRITE_KEY
ENV NUXT_PUBLIC_RUDDERSTACK_WRITE_KEY=$NUXT_PUBLIC_RUDDERSTACK_WRITE_KEY
ARG NUXT_PUBLIC_RUDDERSTACK_DATA_PLANE_URL
ENV NUXT_PUBLIC_RUDDERSTACK_DATA_PLANE_URL=$NUXT_PUBLIC_RUDDERSTACK_DATA_PLANE_URL
RUN pnpm generate

FROM golang AS gobuild
WORKDIR /app
COPY backend/go.mod .
COPY backend/go.sum .
RUN go mod download
COPY backend .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server/

FROM gcr.io/distroless/static
COPY --from=build /app/.output/public /static
COPY --from=gobuild /server /server
EXPOSE 8080
ENTRYPOINT [ "/server" ]
