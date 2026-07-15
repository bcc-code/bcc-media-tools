package main

import (
	"bcc-media-tools/api/v1/apiv1connect"
	"bcc-media-tools/bmm"
	"bcc-media-tools/editorial"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bcc-code/bcc-media-flows/services/cantemo"
	"github.com/bcc-code/bcc-media-flows/services/vidispine/vsapi"
	"github.com/bcc-code/mediabank-bridge/log"
	"github.com/rs/zerolog"

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
	ShortsAPI
	ExportAPI
	CantemoAPI
	VaultAPI
	EditorialAPI
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

	log.ConfigureGlobalLogger(zerolog.DebugLevel)

	if os.Getenv("DEBUG_AUTH_EMAIL") != "" {
		fmt.Printf("DEBUG_AUTH_EMAIL: %s\n", os.Getenv("DEBUG_AUTH_EMAIL"))
	}

	bmmToken, err := bmm.NewBMMToken(
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
		log.L.Debug().Str("temp_path", tempPath).Msg("TEMP_PATH not set, using random path")
	}

	vidispineClient := vsapi.NewClient(
		os.Getenv("VIDISPINE_BASE_URL"),
		os.Getenv("VIDISPINE_USERNAME"),
		os.Getenv("VIDISPINE_PASSWORD"),
	)

	permissionsApi := PermissionsAPI{}
	bmmApi := NewBMMApi(os.Getenv("BMM_BASE_URL"), bmmToken)
	transcriptionAPI := NewTranscriptionAPI(os.Getenv("CANTEMO_URL"), os.Getenv("CANTEMO_TOKEN"), temporalClient)

	// Shared Cantemo client for tool previews and the VAULT proxy handlers.
	cantemoClient := cantemo.NewClient(os.Getenv("CANTEMO_URL"), os.Getenv("CANTEMO_TOKEN"))

	shortsAPI := NewShortsAPI(temporalClient, cantemoClient)
	exportAPI := NewExportAPI(vidispineClient, temporalClient)
	cantemoAPI := NewCantemoAPI(temporalClient)
	vaultAPI := NewVaultAPI(
		vidispineClient,
		os.Getenv("VIDISPINE_BASE_URL"),
		os.Getenv("VIDISPINE_USERNAME"),
		os.Getenv("VIDISPINE_PASSWORD"),
	)

	editorialDBPath := os.Getenv("EDITORIAL_DB_PATH")
	if editorialDBPath == "" {
		editorialDBPath = filepath.Join(os.Getenv("CONFIG_ROOT"), "editorial.db")
	}
	editorialStore, err := editorial.Open(editorialDBPath)
	if err != nil {
		panic(err)
	}
	defer editorialStore.Close()
	editorialAPI := NewEditorialAPI(editorialStore, vidispineClient, cantemoClient)

	api := &ApiServer{
		PermissionsAPI:   permissionsApi,
		BMMApi:           *bmmApi,
		TranscriptionAPI: *transcriptionAPI,
		ShortsAPI:        *shortsAPI,
		ExportAPI:        *exportAPI,
		CantemoAPI:       *cantemoAPI,
		VaultAPI:         *vaultAPI,
		EditorialAPI:     *editorialAPI,
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

	// VAULT media proxies (auth'd, server-side fetch — never expose upstream URLs).
	mux.Handle("/vault/thumbnail", newVaultThumbnailHandler(vaultAPI))
	mux.Handle("/vault/preview", vaultPreviewHandler{
		cantemo:      cantemoClient,
		cantemoToken: os.Getenv("CANTEMO_TOKEN"),
	})
	mux.Handle("/vault/image", newVaultImageHandler(cantemoClient, os.Getenv("CANTEMO_TOKEN")))
	mux.Handle("/vault/waveform", newVaultWaveformHandler(vaultAPI))

	mux.Handle("/subtitle-style-preview", subtitleStylePreviewHandler{})
	mux.Handle("/overlay-preview", overlayPreviewHandler{})

	mux.Handle("/", http.HandlerFunc(serveFiles))

	log.L.Debug().Msg("Starting server on http://localhost:8080/")

	err = http.ListenAndServe(":8080",
		h2c.NewHandler(mux, &http2.Server{}),
	)

	if err != nil {
		log.L.Error().Err(err).Msg("Error starting server")
	}
}

func serveFiles(w http.ResponseWriter, r *http.Request) {
	localPath := filepath.Join(staticFilePath, filepath.Clean(r.URL.Path))

	// SPA fallback: when the requested file doesn't exist (or is a directory),
	// serve index.html for client-side routes — paths without a file extension
	// such as /vault, /vault/, or /vault/123 — so they resolve instead of
	// 404ing. Real assets (.js/.css/images) are still served as-is and 404 when
	// genuinely missing.
	if info, err := os.Stat(localPath); err != nil || info.IsDir() {
		if filepath.Ext(localPath) == "" {
			localPath = filepath.Join(staticFilePath, "index.html")
		}
	}

	http.ServeFile(w, r, localPath)
}
