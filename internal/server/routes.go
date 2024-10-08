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
	mux.HandleFunc("GET /files", s.getVideoIdList)
	mux.HandleFunc("GET /downloadMedia", s.downloadVideoFile)

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
	filepath := fmt.Sprintf("%v%v%s", UPLOAD_DIRECTORY, fileId, filepath.Ext(fileHeader.Filename))
	dst, err := os.Create(filepath)
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

	// Get the currently logged in loggedInUser. Associate the file to the user
	loggedInUser := s.db.GetUserByEmail(getLoggedInUser())
	s.db.MediaUpload(fileId, loggedInUser.Id)

	fmt.Fprintf(w, "Upload successful\n")
}

// TODO: Once there is a basic front end for authentication, how will I store the currently logged in user within the session state?
func getLoggedInUser() string {
	return "test@gmail.com"
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

func (s *Server) downloadVideoFile(w http.ResponseWriter, r *http.Request) {
	type getVideo struct {
		VideoId uuid.UUID `json:"videoId"`
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

	var video getVideo
	if err := json.Unmarshal(body, &video); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Create filepath using the passed in videoId
	mediaFilePath := UPLOAD_DIRECTORY + video.VideoId.String()

	// open the file
	file, err := os.Open(mediaFilePath)
	if err != nil {
		http.Error(w, "File not found.", 404)
		return
	}

	defer file.Close()

	// Get file's Content-Type for correct response header
	mimeType := "application/octet-stream"
	w.Header().Set("Content-Type", mimeType)

	// Set Content-Disposition header to download file
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(mediaFilePath))

	// Serve the file
	http.ServeFile(w, r, mediaFilePath)
}

//TODO: Endpoint for retreiving user information
func (s *Server) getUserInformation(w http.ResponseWriter, r *http.Request) {

}

// Return the list of video id's and storage paths for a requested user
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

	fileIds, err := json.Marshal(s.db.GetMediaListByUserEmail(user.Email))
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(fileIds)
}

// WARN: Got this from an article. Double check
// https://medium.com/@anajankow/fast-check-if-all-struct-fields-are-set-in-golang-bba1917213d2
func ValidateStruct(s interface{}) (err error) {
	// make sure that the input is a struct
	structType := reflect.TypeOf(s)
	if structType.Kind() != reflect.Struct {
		return errors.New("input param should be a struct")
	}

	// go one by one through the fields and validate their value
	structVal := reflect.ValueOf(s)
	fieldNum := structVal.NumField()

	for i := 0; i < fieldNum; i++ {
		field := structVal.Field(i)
		fieldName := structType.Field(i).Name

		// CAREFUL! IsZero interprets empty strings and int equal 0 as a zero value.
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
