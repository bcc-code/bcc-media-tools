api: backend/api/v1/api.pb.go frontend/src/gen/api/v1/api_pb.ts

backend/api/v1/api.pb.go: api/v1/api.proto
	buf generate
