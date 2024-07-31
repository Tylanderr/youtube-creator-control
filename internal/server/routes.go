package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/a-h/templ"
	"github.com/tylander732/youtube-creator-control/cmd/web"
)

// 1mb
const MAXIMUM_FILE_SIZE = 1024 * 1024

type Request struct {
	// Metadata Metadata `json:"metadata"`
	Data string `json:"data"`
	// Add functionality to support video files
}

type Metadata struct {
	// Some stuff related to the video file that's being uploaded
	// Maybe community post as well?
	UploadDate string
	Filename   string
	Filesize   string
}

// Pointer to Server attachs the method to the server struct, and also states
// that modification to the struct will happen
func (s *Server) RegisterRoutes() http.Handler {

	mux := http.NewServeMux()

	// HelloWorldHandler becomes bound to the receiver s, when HandleFunc is called.
	// So the ResponseWriter and Request parameters are implicitly passed
	// See NewServer() call in server.go
	mux.HandleFunc("/", s.HelloWorldHandler)
	mux.HandleFunc("/health", s.healthHandler)

	mux.HandleFunc("POST /post", s.UploadData)

	fileServer := http.FileServer(http.FS(web.Files))

	mux.Handle("/assets/", fileServer)
	mux.Handle("/web", templ.Handler(web.HelloForm()))
	mux.HandleFunc("/hello", web.HelloWebHandler)

	return mux
}

func (s *Server) HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	jsonResp, err := json.Marshal(s.db.Health())

	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

// TODO: Create an endpoint that will upload some data
// And write it to the database
func (s *Server) UploadData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}

	// Close the request body once the processing has finished
	defer r.Body.Close()

	var data Request
	if err := json.Unmarshal(body, &data); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	fmt.Printf("Recieved data: %+v\n", data)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Unable to encode response", http.StatusInternalServerError)
	}

	// TODO: Pass user request data over to database
}

func (s *Server) UploadVideoFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
	
	r.Body = http.MaxBytesReader(w, r.Body, MAXIMUM_FILE_SIZE)
	if err := r.ParseMultipartForm(MAXIMUM_FILE_SIZE); err != nil {
		http.Error(w, "The uploaded file is too big.", http.StatusBadRequest)
		return
	}

	// The argument to FormFile must match the name attribute
	// of the file input on the frontend
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Close file when processing finishes
	defer file.Close()

	err = os.MkdirAll("./uploads", os.ModePerm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create new file in the upload directory
	dst, err := os.Create(fmt.Sprintf("./uploads/%d%s", time.Now().UnixNano(), filepath.Ext(fileHeader.Filename)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer dst.Close()

	// Copy the uploaded file to the filesystem
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

//TODO: Create endpoint that will return data