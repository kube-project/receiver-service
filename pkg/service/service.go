package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"

	"github.com/kube-project/receiver-service/models"
	"github.com/kube-project/receiver-service/pkg/providers"
)

// Config is everything that this service needs to work.
type Config struct {
	Nsq struct {
		Address string
	}
	Producer struct {
		Address string
	}
}

// Dependencies are providers which this service operates with.
type Dependencies struct {
	ImageProvider providers.ImageProvider
	SendProvider  providers.SendProvider
	Logger        zerolog.Logger
}

// Receiver represents the service object of the receiver.
type Receiver struct {
	config Config
	deps   Dependencies
}

// Path is a single path of an image to process.
type Path struct {
	Path string `json:"path"`
}

// Paths is a batch of paths to process.
type Paths struct {
	Paths []Path `json:"paths"`
}

// PostImage handles a post of an image. Saves it to the database
// and sends it to NSQ for further processing.
func (s *Receiver) postImage(w http.ResponseWriter, r *http.Request) {
	var p Path
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, fmt.Sprintf("got error while decoding body: %s", err), http.StatusInternalServerError)
		return
	}

	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Println("failed to close body: ", err)
		}
	}()

	if p.Path == "" {
		http.Error(w, "path cannot be an empty string", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "got path: %+v\n", p)

	ps := Paths{
		Paths: []Path{p},
	}
	var pathsJSON bytes.Buffer

	if err := json.NewEncoder(&pathsJSON).Encode(ps); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode paths: %s", err), http.StatusInternalServerError)
		return
	}

	clone := r.Clone(context.Background())
	clone.Body = io.NopCloser(&pathsJSON)
	clone.ContentLength = int64(pathsJSON.Len())

	s.postImages(w, clone)
}

// PostImages handles a post of multiple images.
func (s *Receiver) postImages(w http.ResponseWriter, r *http.Request) {
	s.deps.Logger.Debug().Msg("post images called...")
	var p Paths
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, fmt.Sprintf("got error while decoding request body: %s", err), http.StatusBadRequest)
		return
	}

	defer func() {
		if err := r.Body.Close(); err != nil {
			s.deps.Logger.Error().Err(err).Msg("failed to close body")
		}
	}()

	if len(p.Paths) == 0 {
		http.Error(w, "paths cannot be empty", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "got paths: %+v with length: %d\n", p, len(p.Paths))
	for _, path := range p.Paths {
		image := models.Image{
			ID:       -1,
			PersonID: -1,
			Path:     []byte(path.Path),
			Status:   models.PENDING,
		}

		savedImage, err := s.deps.ImageProvider.SaveImage(&image)
		if err != nil {
			fmt.Fprintf(w, "got error while saving image: %s; moving on to next...", err)
			continue
		}

		fmt.Fprintf(w, "image saved with id: %d\n", savedImage.ID)

		if err := s.deps.SendProvider.SendImage(uint64(savedImage.ID)); err != nil {
			fmt.Fprintf(w, "error while sending image to queue: %s", err)
			continue
		}

		fmt.Fprintln(w, "image sent to nsq")
	}
}

// New creates a new service will all its needed configuration.
func New(cfg Config, deps Dependencies) *Receiver {
	s := &Receiver{
		config: cfg,
		deps:   deps,
	}
	return s
}

// Run starts this service.
func (s *Receiver) Run(ctx context.Context) error {
	s.deps.Logger.Info().Msg("starting receiver service....")
	router := mux.NewRouter()
	router.HandleFunc("/image/post", s.postImage).Methods("POST")
	router.HandleFunc("/images/post", s.postImages).Methods("POST")
	return http.ListenAndServe(":8000", router)
}
