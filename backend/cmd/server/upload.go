package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	ingestworkflows "github.com/bcc-code/bcc-media-flows/workflows/ingest"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
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
	p := PermissionsForEmail(getEmailFromHttp(r))

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
		if err != nil {
			http.Error(w, "error reading multipart", http.StatusInternalServerError)
			return
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
		filePath = filepath.Join(u.TempPath, uuid.New().String()+filepath.Ext(part.FileName()))

		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "error creating file", http.StatusInternalServerError)
			return
		}

		if _, err := io.Copy(dst, part); err != nil {
			http.Error(w, "error writing file", http.StatusInternalServerError)
			_ = dst.Close()
			return
		}

		_ = dst.Close()
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

	targetEnvironment := formData["environment"]
	if targetEnvironment == "int" {
		targetEnvironment = "bmm-int"
	}

	_, err = u.TemporalClient.ExecuteWorkflow(r.Context(), workflowOptions, ingestworkflows.BmmIngestUpload, ingestworkflows.BmmSimpleUploadParams{
		Title:               formData["title"],
		Language:            convertBMMLanguageCodeToMB(formData["file_language"]),
		TrackID:             trackID,
		UploadedBy:          getEmailFromHttp(r),
		FilePath:            filePath,
		BmmTargetEnvionment: targetEnvironment,
	})

	if err != nil {
		http.Error(w, "error starting workflow", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// The table is based on
// https://github.com/bcc-code/bmm-api/blob/develop/BMM.Api.Core/BtvLanguageProvider.cs
var bmmToMB = map[string]string{
	"nb":  "nor",
	"de":  "deu",
	"nl":  "nld",
	"fr":  "fra",
	"ru":  "rus",
	"ro":  "ron",
	"pl":  "pol",
	"bg":  "bul",
	"hu":  "hun",
	"sl":  "slv",
	"hr":  "hrv",
	"tr":  "tur",
	"en":  "eng",
	"es":  "spa",
	"it":  "ita",
	"pt":  "por",
	"fi":  "fin",
	"zh":  "cmn",
	"da":  "dan",
	"yue": "yue",
	"ml":  "mal",
	"ta":  "tam",
	"et":  "est",
	"kha": "kha",
	"af":  "af",
}

func convertBMMLanguageCodeToMB(lang string) string {
	if bmmLang, ok := bmmToMB[lang]; ok {
		return bmmLang
	}

	// If it's not a bmm language, return it as is
	// this is better than to fail at this point, and it can be corrected manually later if needed
	return lang
}

var langToFlagFile = map[string]string{
	"nb":  "no.svg",
	"de":  "de.svg",
	"nl":  "nl.svg",
	"fr":  "fr.svg",
	"ru":  "ru.svg",
	"ro":  "ro.svg",
	"pl":  "pl.svg",
	"bg":  "bg.svg",
	"hu":  "hu.svg",
	"sl":  "si.svg",
	"hr":  "hr.svg",
	"tr":  "tr.svg",
	"en":  "gb.svg",
	"es":  "es.svg",
	"it":  "it.svg",
	"pt":  "pt.svg",
	"fi":  "fi.svg",
	"zh":  "cn.svg",
	"da":  "dk.svg",
	"yue": "hk.svg",
	"ml":  "in.svg",
	"ta":  "in.svg",
	"et":  "ee.svg",
	"kha": "in.svg",
	"af":  "za.svg",
}

func IconForLang(lang string) string {
	if iconPath, ok := langToFlagFile[lang]; ok {
		return iconPath
	}
	return "xx.svg"
}
