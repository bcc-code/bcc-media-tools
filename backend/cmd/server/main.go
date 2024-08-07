package main

import (
	"bcc-media-tools/api/v1/apiv1connect"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"go.temporal.io/sdk/client"

	connectcors "connectrpc.com/cors"

	"connectrpc.com/connect"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// EmailHeader is added by the Proxy server
//
// The server handles all authentication, so we can trust that the email is authenticated,
// and we can use it to look up permissions.
const EmailHeader = "x-token-user-email"

var staticFilePath = "/static/"

func getEmailFromHttp(r *http.Request) string {
	if e := os.Getenv("DEBUG_AUTH_EMAIL"); e != "" {
		return e
	}

	return r.Header.Get(EmailHeader)
}

func getEmail[T any](req *connect.Request[T]) string {
	if e := os.Getenv("DEBUG_AUTH_EMAIL"); e != "" {
		return e
	}

	return req.Header().Get(EmailHeader)
}

type ApiServer struct {
	PermissionsAPI
	BMMApi
	TranscriptionAPI
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

func NewTemporalClient(host, namespace string) (client.Client, error) {
	return client.Dial(client.Options{
		HostPort:  host,
		Namespace: namespace,
	})
}

func main() {
	_ = godotenv.Load()

	bmmToken, err := NewBMMToken(
		os.Getenv("BMM_AUTH0_BASE_URL"),
		os.Getenv("BMM_CLIENT_ID"),
		os.Getenv("BMM_CLIENT_SECRET"),
		os.Getenv("BMM_AUDIENCE"),
	)

	if err != nil {
		panic(err)
	}

	temporalClient, err := NewTemporalClient(
		os.Getenv("TEMPORAL_HOST_PORT"),
		os.Getenv("TEMPORAL_NAMESPACE"),
	)

	if err != nil {
		panic(err)
	}

	tempPath := os.Getenv("TEMP_PATH")
	if tempPath == "" {
		tempPath = os.TempDir()
		fmt.Printf("TEMP_PATH not set, using %s\n", tempPath)
	}

	permissionsApi := PermissionsAPI{}
	bmmApi := NewBMMApi(os.Getenv("BMM_BASE_URL"), bmmToken)
	transcriptionAPI := NewTranscriptionAPI(os.Getenv("CANTEMO_URL"), os.Getenv("CANTEMO_TOKEN"))

	api := &ApiServer{
		PermissionsAPI:   permissionsApi,
		BMMApi:           *bmmApi,
		TranscriptionAPI: *transcriptionAPI,
	}

	if os.Getenv("STATIC_FILE_PATH") != "" {
		staticFilePath = os.Getenv("STATIC_FILE_PATH")
	}

	path, handler := apiv1connect.NewAPIServiceHandler(api)

	handler = withCORS(handler)

	mux := http.NewServeMux()
	mux.Handle(path, handler)
	mux.Handle("/upload", uploadHandler{
		TemporalClient: temporalClient,
		TempPath:       tempPath,
	})

	mux.Handle("/", http.HandlerFunc(serveFiles))

	_ = http.ListenAndServe(":8080",
		h2c.NewHandler(mux, &http2.Server{}),
	)
}

func serveFiles(w http.ResponseWriter, r *http.Request) {
	localPath := filepath.Join(staticFilePath, r.URL.Path)

	if r.URL.Path[len(r.URL.Path)-1] == '/' {
		localPath = filepath.Join(staticFilePath, "/index.html")
	}

	http.ServeFile(w, r, localPath)
}
