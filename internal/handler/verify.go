package handler

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
)

// VerifyFiles checks the integrity of all synced files by comparing
// their local MD5 checksums against the stored Drive checksums.
func (h *Handler) VerifyFiles(w http.ResponseWriter, r *http.Request) {
	files, err := h.Store.ListFiles("")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Count total files to verify (non-folder)
	total := 0
	for _, f := range files {
		if !f.IsFolder {
			total++
		}
	}

	var resp VerifyResponse
	checked := 0
	for _, f := range files {
		if f.IsFolder {
			continue
		}
		resp.Total++
		checked++

		if f.MD5Checksum == "" {
			resp.Skipped++
			resp.Results = append(resp.Results, VerifyResult{
				FileID: f.FileID,
				Name:   f.Name,
				Status: "skipped",
			})
		} else if actual, err := fileMD5(f.LocalPath); err != nil {
			resp.Missing++
			resp.Results = append(resp.Results, VerifyResult{
				FileID:   f.FileID,
				Name:     f.Name,
				Status:   "missing",
				Expected: f.MD5Checksum,
			})
		} else if actual != f.MD5Checksum {
			resp.Mismatch++
			resp.Results = append(resp.Results, VerifyResult{
				FileID:   f.FileID,
				Name:     f.Name,
				Status:   "mismatch",
				Expected: f.MD5Checksum,
				Actual:   actual,
			})
		} else {
			resp.Ok++
			resp.Results = append(resp.Results, VerifyResult{
				FileID: f.FileID,
				Name:   f.Name,
				Status: "ok",
			})
		}

		// Broadcast progress via WebSocket
		if h.Hub != nil {
			msg, _ := json.Marshal(map[string]any{
				"type": "verify_progress",
				"data": map[string]any{
					"checked":  checked,
					"total":    total,
					"fileName": f.Name,
				},
			})
			h.Hub.Broadcast(msg)
		}
	}

	// Broadcast completion
	if h.Hub != nil {
		msg, _ := json.Marshal(map[string]any{
			"type": "verify_complete",
			"data": map[string]any{
				"total":    resp.Total,
				"ok":       resp.Ok,
				"mismatch": resp.Mismatch,
				"missing":  resp.Missing,
				"skipped":  resp.Skipped,
			},
		})
		h.Hub.Broadcast(msg)
	}

	writeJSON(w, http.StatusOK, resp)
}


// ResyncFiles clears checksums for the given file IDs so the next
// full sync will re-download them, then starts a full sync.
func (h *Handler) ResyncFiles(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FileIDs []string `json:"file_ids"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if len(req.FileIDs) == 0 {
		writeError(w, http.StatusBadRequest, "no file IDs provided")
		return
	}

	cleared, err := h.Store.ClearFileChecksums(req.FileIDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"cleared": cleared,
	})
}

func fileMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
