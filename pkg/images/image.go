package images

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog"

	"github.com/kube-project/receiver-service/models"
)

// Config db configs here
type Config struct {
	Port             string
	Dbname           string
	UsernamePassword string
	Hostname         string
	Logger           zerolog.Logger
}

type imageProvider struct {
	config Config
}

// NewImageProvider creates a new image provider using the db.
func NewImageProvider(cfg Config) *imageProvider {
	return &imageProvider{config: cfg}
}

// SaveImage takes an image model and saves it into the database.
func (i *imageProvider) SaveImage(image *models.Image) (*models.Image, error) {
	i.config.Logger.Debug().Str("path", string(image.Path)).Msg("Saving image path...")

	var (
		result sql.Result
		err    error
	)
	f := func(tx *sql.Tx) error {
		result, err = tx.Exec("insert into images (path, person, status) values (?, ?, ?)", image.Path, image.PersonID, image.Status)
		if err != nil {
			i.config.Logger.Debug().Err(err).Msg("failed to run insert")
			return fmt.Errorf("failed to insert image: %w", err)
		}

		return nil
	}

	if err := i.execInTx(context.Background(), f); err != nil {
		return nil, fmt.Errorf("failed to run transaction: %w", err)
	}

	id, _ := result.LastInsertId()
	image.ID = id

	return image, nil
}

// LoadImage takes an id and looks for that id in the database.
func (i *imageProvider) LoadImage(id int64) (*models.Image, error) {
	i.config.Logger.Info().Int64("id", id).Msg("Loading image with ID")

	var (
		imageID int
		path    string
		person  int
		status  int
	)
	f := func(tx *sql.Tx) error {
		if err := tx.QueryRow("select id, path, person, status from images where id = ?", id).Scan(&imageID, &path, &person, status); err != nil {
			return fmt.Errorf("failed to run query: %w", err)
		}

		return nil
	}

	if err := i.execInTx(context.Background(), f); err != nil {
		return nil, fmt.Errorf("failed to run query in transaction: %w", err)
	}

	ret := &models.Image{
		ID:       int64(imageID),
		Path:     []byte(path),
		PersonID: person,
		Status:   models.Status(status),
	}
	return ret, nil
}

// execInTx executes in transaction. It will either commit, or rollback if there was an error.
func (i *imageProvider) execInTx(ctx context.Context, f func(tx *sql.Tx) error) error {
	db, err := i.connect()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	// Defer a rollback in case anything fails.
	defer func() {
		if err := tx.Rollback(); err != nil {
			log.Println("Failed to rollback: ", err)
		}
	}()
	defer func() {
		if err := db.Close(); err != nil {
			log.Println("Failed to close database: ", err)
		}
	}()

	if err := f(tx); err != nil {
		return fmt.Errorf("failed to run function in transaction: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (i *imageProvider) createConnectionString() string {
	return fmt.Sprintf("%s@tcp(%s:%s)/%s",
		i.config.UsernamePassword,
		i.config.Hostname,
		i.config.Port,
		i.config.Dbname)
}

func (i *imageProvider) connect() (*sql.DB, error) {
	return sql.Open("mysql", i.createConnectionString())
}
