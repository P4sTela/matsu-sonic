package handler

import (
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/P4sTela/matsu-sonic/internal/config"
	"github.com/P4sTela/matsu-sonic/internal/drive"
	"github.com/P4sTela/matsu-sonic/internal/store"
	msync "github.com/P4sTela/matsu-sonic/internal/sync"
)

// Handler holds shared dependencies for all endpoint handlers.
type Handler struct {
	Config     *config.Config
	ConfigPath string
	Store      *store.DB
	Drive      *drive.DriveClient
	Engine     *msync.SyncEngine
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
