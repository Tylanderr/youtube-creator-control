package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
	"github.com/tylanderr/youtube-creator-control/internal/structs"
)

// Service represents a service that interacts with a database.
type Service interface {
	// Health returns a map of health status information.
	// The keys and values in the map are service-specific.
	Health() map[string]string

	// Write to the DB
	AddNewUser(addUser structs.AddUser) map[string]string
	MediaUpload(fileId uuid.UUID, userId uuid.UUID) map[string]string
	GetUserByEmail(email string) User
	GetMediaListByUserEmail(email string) []uuid.UUID

	// Close terminates the database connection.
	// It returns an error if the connection cannot be closed.
	Close() error
}

type service struct {
	db *sql.DB
}

type User struct {
	Id        uuid.UUID
	Email     string
	FirstName string
	LastName  string
}

var (
	database   = os.Getenv("DB_DATABASE")
	password   = os.Getenv("DB_PASSWORD")
	username   = os.Getenv("DB_USERNAME")
	port       = os.Getenv("DB_PORT")
	host       = os.Getenv("DB_HOST")
	schema     = os.Getenv("DB_SCHEMA")
	dbInstance *service
)

func New() Service {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance
	}
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s", username, password, host, port, database, schema)
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal(err)
	}
	dbInstance = &service{
		db: db,
	}
	return dbInstance
}

// Health checks the health of the database connection by pinging the database.
// It returns a map with keys indicating various health statistics.
func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	// Ping the database
	err := s.db.PingContext(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Fatalf(fmt.Sprintf("db down: %v", err)) // Log the error and terminate the program
		return stats
	}

	// Database is up, add more statistics
	stats["status"] = "up"
	stats["message"] = "It's healthy"

	// Get database stats (like open connections, in use, idle, etc.)
	dbStats := s.db.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	// Evaluate stats to provide a health message
	if dbStats.OpenConnections > 40 { // Assuming 50 is the max for this example
		stats["message"] = "The database is experiencing heavy load."
	}

	if dbStats.WaitCount > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

// Close closes the database connection.
// It logs a message indicating the disconnection from the specific database.
// If the connection is successfully closed, it returns nil.
// If an error occurs while closing the connection, it returns the error.
func (s *service) Close() error {
	log.Printf("Disconnected from database: %s", database)
	return s.db.Close()
}

// Write new user to database
func (s *service) AddNewUser(addUser structs.AddUser) map[string]string {

	status := make(map[string]string)

	query := `INSERT INTO users (email, first_name, last_name) VALUES ($1, $2, $3)`
	_, err := s.db.Exec(query, addUser.Email, addUser.FirstName, addUser.LastName)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Writing to database: %s", database)
	status["write_successful"] = "true"

	return status
}


func (s *service) GetUserByEmail(email string) User {
	query := `SELECT * FROM users WHERE email = $1`

	var user User

	err := s.db.QueryRow(query, email).Scan(&user.Id, &user.Email, &user.FirstName, &user.LastName)

	if err != nil {
		// If the error is noRows, it means the email provided is not in use.
		// Return empty User struct
		if err == sql.ErrNoRows {
			fmt.Println(err)
			return User{}
		} else {
			log.Fatalf("Query failed: %v", err)
		}
	}

	return user
}


func (s *service) GetMediaListByUserEmail(email string) []uuid.UUID {
	query := `SELECT media.file_id FROM media JOIN users ON media.user_id = users.id WHERE users.email = $1`

	fileIds := []uuid.UUID{}

	rows, err := s.db.Query(query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println(err)
			return nil
		} else {
			log.Fatalf("Query failed: %v", err)
		}
	}

	defer rows.Close()

	for rows.Next() {
		var file_id uuid.UUID

		err := rows.Scan(&file_id)
		if err != nil {
			log.Fatal(err)
		}

		fileIds = append(fileIds, file_id)

	}

	return fileIds
}

func (s *service) MediaUpload(fileId uuid.UUID, userId uuid.UUID) map[string]string {
	status := make(map[string]string)

	query := `INSERT INTO media (file_id, user_id) VALUES ($1, $2)`
	_, err := s.db.Exec(query, fileId, userId)

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Writing to database: %s", database)
	status["write_successful"] = "true"

	return status
}
