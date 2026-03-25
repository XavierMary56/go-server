package admin

import (
	"encoding/json"
	"net/http"
)

func (ah *AdminHandler) jsonOK(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (ah *AdminHandler) jsonError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":  status,
		"error": msg,
	})
}
