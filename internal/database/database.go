package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/copybridge/copybridge-server/internal/clipboard"

	_ "github.com/joho/godotenv/autoload"
	_ "github.com/mattn/go-sqlite3"
)

// Service represents a service that interacts with a database.
type Service interface {
	// Health returns a map of health status information.
	// The keys and values in the map are service-specific.
	Health() map[string]string

	// Insert inserts a new clipboard into the database.
	// It returns an error if the insertion fails.
	Insert(c *clipboard.Clipboard) error

	// Get retrieves a clipboard from the database by its id.
	// It returns nil if the clipboard does not exist.
	// It returns an error if the retrieval fails.
	Get(id int) (*clipboard.Clipboard, error)

	// Update updates an existing clipboard in the database.
	// It returns an error if the update fails.
	Update(c *clipboard.Clipboard) error

	// Delete deletes a clipboard from the database by its id.
	// It returns an error if the deletion fails.
	Delete(id int) error

	// Close terminates the database connection.
	// It returns an error if the connection cannot be closed.
	Close() error
}

type service struct {
	db *sql.DB
}

var (
	dburl      = os.Getenv("DB_URL")
	dbInstance *service
)

func New() Service {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance
	}

	db, err := sql.Open("sqlite3", dburl)
	if err != nil {
		// This will not be a connection error, but a DSN parse error or
		// another initialization error.
		log.Fatal(err)
	}

	// Check if table exists
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='clipboards';")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// If table does not exist, create it
	if !rows.Next() {
		_, err = db.Exec(`CREATE TABLE clipboards (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			data TEXT NOT NULL,
			is_encrypted BOOLEAN NOT NULL DEFAULT FALSE,
			password_hash TEXT,
			salt TEXT,
			nonce TEXT
		);`)
		if err != nil {
			log.Fatal(err)
		}

		// Insert a row with id 99999
		_, err = db.Exec(`INSERT INTO clipboards (id, name, type, data) VALUES (?, ?, ?, ?);`, 99999, "example", "text/plain", "Hello, World!")
		if err != nil {
			log.Fatal(err)
		}

		// Delete the row with id 99999
		_, err = db.Exec(`DELETE FROM clipboards WHERE id = ?;`, 99999)
		if err != nil {
			log.Fatal(err)
		}
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
	log.Printf("Disconnected from database: %s", dburl)
	return s.db.Close()
}

// Insert inserts a new clipboard into the database.
// If the clipboard is encrypted, it inserts the encrypted data along with the password hash, salt, and nonce.
// If the clipboard is not encrypted, it inserts the data as is.
// If the insertion is successful, it returns nil.
// If an error occurs during insertion, it returns the error.
func (s *service) Insert(c *clipboard.Clipboard) error {
	sqlInsert := `INSERT INTO clipboards (name, type, data) VALUES (?, ?, ?);`
	sqlInsertEncrypted := `INSERT INTO clipboards (name, type, data, is_encrypted, password_hash, salt, nonce) VALUES (?, ?, ?, ?, ?, ?, ?);`

	var result sql.Result
	var err error
	if c.IsEncrypted {
		result, err = s.db.Exec(sqlInsertEncrypted, c.Name, c.DataType, c.Data, c.IsEncrypted, c.PasswordHash, c.Salt, c.Nonce)
	} else {
		result, err = s.db.Exec(sqlInsert, c.Name, c.DataType, c.Data)
	}
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	c.Id = int(id)

	return nil
}

// Get retrieves a clipboard from the database by its id.
// If the clipboard is encrypted, it retrieves the encrypted data along with the password hash, salt, and nonce.
// If the clipboard is not encrypted, it retrieves the data as is.
// If the clipboard does not exist, it returns nil.
// If an error occurs during retrieval, it returns the error.
func (s *service) Get(id int) (*clipboard.Clipboard, error) {
	sqlSelect := `SELECT * FROM clipboards WHERE id = ?;`

	var c clipboard.Clipboard
	var passwordHash, salt, nonce sql.NullString
	err := s.db.QueryRow(sqlSelect, id).
		Scan(&c.Id, &c.Name, &c.DataType, &c.Data, &c.IsEncrypted, &passwordHash, &salt, &nonce)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if c.IsEncrypted {
		c.PasswordHash = passwordHash.String
		c.Salt = salt.String
		c.Nonce = nonce.String
	}

	return &c, nil
}

// Update updates an existing clipboard in the database.
// If
func (s *service) Update(c *clipboard.Clipboard) error {
	sqlUpdate := `UPDATE clipboards SET name = ?, type = ?, data = ?, nonce = ? WHERE id = ?;`

	_, err := s.db.Exec(sqlUpdate, c.Name, c.DataType, c.Data, c.Nonce, c.Id)
	return err
}

// Delete deletes a clipboard from the database by its id.
func (s *service) Delete(id int) error {
	sqlDelete := `DELETE FROM clipboards WHERE id = ?;`

	_, err := s.db.Exec(sqlDelete, id)
	return err
}
