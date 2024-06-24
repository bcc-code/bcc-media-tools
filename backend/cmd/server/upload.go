package main

import (
	"github.com/bcc-code/bcc-media-flows/workflows/webhooks"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

type uploadHandler struct {
	TemporalClient client.Client
	TempPath       string
}

func getQueue() string {
	queue := os.Getenv("QUEUE")
	if queue == "" {
		queue = "worker"
	}
	return queue
}

// ServeHTTP handles the upload request
//
// The low level approach is used here inorder to handle the multipart form data
// in a streaming fashion. This is useful for large files.
func (u uploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	enableCors(&w)

	if r.Method == http.MethodOptions {
		return
	}

	// Check permissions
	// Note that this permission check is different as it does not use GRPC
	p := PermissionsForEmail(r.Header.Get("x-token-user-email"))

	if !p.CanUpload() {
		http.Error(w, "permission denied", http.StatusForbidden)
		return
	}
	// End permission check

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "error reading multipart", http.StatusInternalServerError)
		return
	}

	filePath := ""
	formData := map[string]string{}

	// Handle upload
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}

		if part.FileName() == "" { // this is not a file
			data, err := io.ReadAll(part)
			if err != nil {
				http.Error(w, "error reading data", http.StatusInternalServerError)
				return
			}
			formData[part.FormName()] = string(data)
			continue
		}

		// Ext includes the dot
		filePath = filepath.Join("/tmp/", uuid.New().String()+filepath.Ext(part.FileName()))

		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "error creating file", http.StatusInternalServerError)
			return
		}

		defer dst.Close()

		if _, err := io.Copy(dst, part); err != nil {
			http.Error(w, "error writing file", http.StatusInternalServerError)
			return
		}
	}

	// Trigger flow
	queue := getQueue()
	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: queue,
	}

	trackID, err := strconv.Atoi(formData["trackId"])
	if err != nil {
		http.Error(w, "invalid track id", http.StatusBadRequest)
		return
	}

	_, err = u.TemporalClient.ExecuteWorkflow(r.Context(), workflowOptions, webhooks.BmmSimpleUpload, webhooks.BmmSimpleUploadParams{
		Title:      formData["title"],
		Language:   formData["language"],
		TrackID:    trackID,
		UploadedBy: formData["email"],
		FilePath:   filePath,
	})

	if err != nil {
		http.Error(w, "error starting workflow", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
