package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"

	// "github.com/a-h/templ"
	"github.com/google/uuid"
	"github.com/tylanderr/youtube-creator-control/cmd/web"
	"github.com/tylanderr/youtube-creator-control/internal/database"
	"github.com/tylanderr/youtube-creator-control/internal/structs"
)

// 1mb
const MAXIMUM_FILE_SIZE = 1024 * 1024
const UPLOAD_DIRECTORY = "./uploads/"

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
	mux.HandleFunc("POST /postMedia", s.UploadMediaFile)

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

/////////// Post Methods ///////////

func (s *Server) newUserHandler(w http.ResponseWriter, r *http.Request) {
	var addUser structs.AddUser

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

	if err := json.Unmarshal(body, &addUser); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate that all fields for addUser are present
	err = ValidateStruct(addUser)
	if err != nil {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	response := s.db.GetUserByEmail(addUser.Email)
	// If response isn't an empty User struct, then email already in use
	if response != (database.User{}) {
		http.Error(w, "Email addresss already in use", http.StatusBadRequest)
		return
	}

	jsonResp, err := json.Marshal(s.db.AddNewUser(addUser))
	if err != nil {
		log.Fatalf("error handling JSON masrshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)

}

func (s *Server) UploadMediaFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
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

	err = os.MkdirAll(UPLOAD_DIRECTORY, os.ModePerm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fileId := uuid.New()

	// Create new file in the upload directory
	dst, err := os.Create(fmt.Sprintf("%v%v%s", UPLOAD_DIRECTORY, fileId, filepath.Ext(fileHeader.Filename)))
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

	//TODO: Replace this
	// Pass user email when request comes from front end?
	response := s.db.GetUserByEmail("test@gmail.com")

	s.db.MediaUpload(fileId, response.Id)

	fmt.Fprintf(w, "Upload successful\n")
}

/////////// GET Methods ///////////

func (s *Server) getUserHandler(w http.ResponseWriter, r *http.Request) {
	type getUserRequest struct {
		Email string `json:"email"`
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method now allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	var user getUserRequest
	if err := json.Unmarshal(body, &user); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	jsonResp, err := json.Marshal(s.db.GetUserByEmail(user.Email))
	if err != nil {
		log.Fatalf("error handling JSON masrshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

// TODO: Endpoint for retreiving media file
func (s *Server) RetrieveVideoFile(w http.ResponseWriter, r *http.Request) {

}

// TODO: Endpoint for retreiving user information
// List of videos uploaded
// List of shared videos
func (s *Server) getVideoIdList(w http.ResponseWriter, r *http.Request) {
	type getUser struct {
		Email string `json:"email"`
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	var user getUser
	if err := json.Unmarshal(body, &user); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	//TODO: Finish
}

// WARN: Got this from an article. Double check
// https://medium.com/@anajankow/fast-check-if-all-struct-fields-are-set-in-golang-bba1917213d2
func ValidateStruct(s interface{}) (err error) {
	// first make sure that the input is a struct
	// having any other type, especially a pointer to a struct,
	// might result in panic
	structType := reflect.TypeOf(s)
	if structType.Kind() != reflect.Struct {
		return errors.New("input param should be a struct")
	}

	// now go one by one through the fields and validate their value
	structVal := reflect.ValueOf(s)
	fieldNum := structVal.NumField()

	for i := 0; i < fieldNum; i++ {
		// Field(i) returns i'th value of the struct
		field := structVal.Field(i)
		fieldName := structType.Field(i).Name

		// CAREFUL! IsZero interprets empty strings and int equal 0 as a zero value.
		// To check only if the pointers have been initialized,

		// you can check the kind of the field:
		// if field.Kind() == reflect.Pointer { // check }

		// IsZero panics if the value is invalid.
		// Most functions and methods never return an invalid Value.

		isSet := field.IsValid() && !field.IsZero()

		if !isSet {
			err = errors.New(fmt.Sprintf("%v%s in not set; ", err, fieldName))
		}

	}

	return err
}

////////// REFERENCE CODE //////////

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
