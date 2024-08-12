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

	// "github.com/a-h/templ"
	"github.com/tylander732/youtube-creator-control/cmd/web"
)

// 1mb
const MAXIMUM_FILE_SIZE = 1024 * 1024

//TODO: Ensure that the requests I'm receiving are valid and match only acceptable parameters
//TODO: Create structs representing acceptable requests for user creation


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

	// Handlers becomes bound to the receiver s, when HandleFunc is called.
	// So the ResponseWriter and Request parameters are implicitly passed
	// See NewServer() call in server.go
	mux := http.NewServeMux()

	// POST handlers
	mux.HandleFunc("POST /newUser", s.newUserHandler)
	mux.HandleFunc("POST /postMedia", s.UploadVideoFile)

	// GET handlers
	mux.HandleFunc("GET /getUser", s.getUserHandler)


	fileServer := http.FileServer(http.FS(web.Files))
	mux.HandleFunc("/health", s.healthHandler)
	mux.Handle("/assets/", fileServer)
	mux.HandleFunc("/hello", web.HelloWebHandler)

	return mux
}


func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	jsonResp, err := json.Marshal(s.db.Health())

	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

func (s *Server) newUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

	jsonResp, err := json.Marshal(s.db.AddNewUser("test@gmail.com", "Tyler"))

	if err != nil {
		log.Fatalf("error handling JSON masrshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)

}

func (s *Server) getUserHandler(w http.ResponseWriter, r *http.Request) {
	jsonResp, err := json.Marshal(s.db.GetUserByEmail("test@gmail.com"))
	if err != nil {
		log.Fatalf("error handling JSON masrshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

// TODO: Replace hardcoded path locations
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
	// Ex: curl -F "file/=@/media/blah"
	// or: curl -F "image/=@/media/blah"
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Close file when processing finishes
	defer file.Close()

	// Determine content type. Read the first 512 bytes
	buff := make([]byte, 512)
	_, err = file.Read(buff)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// DetectContentType considers at most the first 512 bytes
	filetype := http.DetectContentType(buff)
	if filetype != "image/jpeg" && filetype != "image/png" {
		http.Error(w, "The provided file format is not allowed", http.StatusInternalServerError)
		return
	}

	// When io.Copy is called later, it will continue reading from the point the file was at after
	// reading the 512 bytes. Resulting in a corrupted image file
	// file.Seek() returns the pointer back to the start of the file
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

	fmt.Fprintf(w, "Upload successful \n")

	//TODO: DB Update with fileID and relevant user data
}

//TODO: Endpoint for retreiving media file
func (s *Server) RetrieveVideoFile(w http.ResponseWriter, r *http.Request) {

}

//TODO: Endpoint for retreiving user information
// List of videos uploaded
// List of shared videos
func (s *Server) getVideoIdList(w http.ResponseWriter, r *http.Request) {

}




////////// REFERENCE CODE //////////


func (s *Server) HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

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
}
