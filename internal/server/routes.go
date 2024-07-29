package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/a-h/templ"
	"github.com/tylander732/youtube-creator-control/cmd/web"
)

type Request struct {
	Metadata Metadata `json:"metadata"`
	Data     string   `json:"data"`
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

//TODO: Create endpoint that will return data
