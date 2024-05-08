package main

import (
	"bcc-media-tools/gen/api/v1/apiv1connect"
	"net/http"
	"os"

	connectcors "connectrpc.com/cors"

	"connectrpc.com/connect"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const EmailHeader = "x-token-user-email"

func getEmail[T any](req *connect.Request[T]) string {
	if e := os.Getenv("DEBUG_AUTH_EMAIL"); e != "" {
		return e
	}

	return req.Header().Get(EmailHeader)
}

type ApiServer struct {
	PermissionsAPI
	BMMApi
}

func withCORS(connectHandler http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"}, // replace with your domain
		AllowedMethods: connectcors.AllowedMethods(),
		AllowedHeaders: connectcors.AllowedHeaders(),
		ExposedHeaders: connectcors.ExposedHeaders(),
		MaxAge:         7200, // 2 hours in seconds
	})
	return c.Handler(connectHandler)
}

func main() {
	bmmToken, err := NewBMMToken(
		os.Getenv("BMM_AUTH0_BASE_URL"),
		os.Getenv("BMM_CLIENT_ID"),
		os.Getenv("BMM_CLIENT_SECRET"),
		os.Getenv("BMM_AUDIENCE"),
	)

	if err != nil {
		panic(err)
	}

	permissionsApi := PermissionsAPI{}
	bmmApi := NewBMMApi(os.Getenv("BMM_BASE_URL"), bmmToken)

	api := &ApiServer{
		PermissionsAPI: permissionsApi,
		BMMApi:         *bmmApi,
	}

	path, handler := apiv1connect.NewAPIServiceHandler(api)

	handler = withCORS(handler)

	mux := http.NewServeMux()
	mux.Handle(path, handler)
	mux.Handle("/upload", uploadHandler{})
	_ = http.ListenAndServe(":8080",
		h2c.NewHandler(mux, &http2.Server{}),
	)
}
