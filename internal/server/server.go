package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"github.com/tylander732/youtube-creator-control/internal/database"
)

type Server struct {
	port int

	db database.Service
}

// Returning a pointer to a struct to allow modification to the structs fields by the caller
// A Server struct has a Handler, which contains ResponseWriters and Requests
func NewServer() *http.Server {
	//Atoi is equivalent to ParseInt. PORT is defined in the .env file
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	NewServer := &Server{
		port: port,

		db: database.New(),
	}

	// Declare Server config
	// Return a pointer to the struct
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
