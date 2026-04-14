package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"reflect"

	"github.com/P4sTela/matsu-sonic/internal/config"
	"github.com/P4sTela/matsu-sonic/internal/distribution"
	"github.com/P4sTela/matsu-sonic/internal/drive"
	"github.com/P4sTela/matsu-sonic/internal/store"
	msync "github.com/P4sTela/matsu-sonic/internal/sync"
)

// Handler holds shared dependencies for all endpoint handlers.
type Handler struct {
	Config      *config.Config
	ConfigPath  string
	Store       *store.DB
	Drive       *drive.DriveClient
	Engine      *msync.SyncEngine
	DistManager *distribution.Manager
}

// ReinitDrive attempts to create a new Drive client from current config.
// Called after config changes that may affect credentials.
func (h *Handler) ReinitDrive() {
	drv, err := drive.NewDriveClient(context.Background(), h.Config)
	if err != nil {
		log.Printf("[handler] Drive client reinit failed: %v", err)
		return
	}
	h.Drive = drv
	h.Engine.SetDriveClient(drv)
	log.Println("[handler] Drive client reinitialized")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	// Ensure nil slices are serialized as [] instead of null
	if v != nil {
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			v = []struct{}{}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
