package admin

import "net/http"

// RegisterRoutes registers admin API routes and the web UI.
func (ah *AdminHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/admin/keys", ah.withAdminAuth(ah.handleProjectKeys))
	mux.HandleFunc("/v1/admin/keys/", ah.withAdminAuth(ah.handleProjectKeyDetail))
	mux.HandleFunc("/v1/admin/health", ah.handleAdminHealth)
	mux.HandleFunc("/v1/admin/keys/check-all", ah.withAdminAuth(ah.handleCheckAllKeys))
	mux.HandleFunc("/v1/admin/anthropic-keys/check", ah.withAdminAuth(ah.handleCheckAnthropicKey))
	mux.HandleFunc("/v1/admin/provider-keys/check", ah.withAdminAuth(ah.handleCheckProviderKey))

	mux.HandleFunc("/v1/admin/projects", ah.withAdminAuth(ah.handleListProjects))
	mux.HandleFunc("/v1/admin/projects/logs", ah.withAdminAuth(ah.handleProjectLogs))
	mux.HandleFunc("/v1/admin/projects/stats", ah.withAdminAuth(ah.handleProjectStats))
	mux.HandleFunc("/v1/admin/settings/admin-token", ah.withAdminAuth(ah.handleAdminTokenSettings))

	mux.HandleFunc("/v1/admin/anthropic-keys", ah.withAdminAuth(ah.handleAnthropicKeys))
	mux.HandleFunc("/v1/admin/anthropic-keys/", ah.withAdminAuth(ah.handleAnthropicKeyDetail))
	mux.HandleFunc("/v1/admin/anthropic-keys/verify", ah.withAdminAuth(ah.handleVerifyAnthropicKey))

	mux.HandleFunc("/v1/admin/provider-keys", ah.withAdminAuth(ah.handleProviderKeys))
	mux.HandleFunc("/v1/admin/provider-keys/", ah.withAdminAuth(ah.handleProviderKeyDetail))

	mux.HandleFunc("/v1/admin/models", ah.withAdminAuth(ah.handleModels))
	mux.HandleFunc("/v1/admin/models/", ah.withAdminAuth(ah.handleModelDetail))

	ah.registerWebUI(mux)
}

func (ah *AdminHandler) handleAdminHealth(w http.ResponseWriter, r *http.Request) {
	ah.jsonOK(w, http.StatusOK, map[string]interface{}{
		"code":                200,
		"status":              "ok",
		"admin_api_available": true,
	})
}
