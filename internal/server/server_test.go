package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danpicton/fauxtes-server/internal/auth"
	"github.com/danpicton/fauxtes-server/internal/storage/sqlite"
)

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	db, err := sqlite.OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })

	srv, err := NewWithDB(db, Config{AuthConfig: auth.DefaultServiceConfig()})
	if err != nil {
		t.Fatalf("NewWithDB() error = %v", err)
	}
	return httptest.NewServer(srv.Mux)
}

func register(t *testing.T, ts *httptest.Server) auth.AuthResponse {
	t.Helper()
	body := `{"email":"test@example.com","password":"test-password","api":"004","pw_nonce":"nonce","version":"004"}`
	resp, err := http.Post(ts.URL+"/v1/users", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("register POST error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register status = %d, want 200", resp.StatusCode)
	}
	var authResp auth.AuthResponse
	json.NewDecoder(resp.Body).Decode(&authResp)
	return authResp
}

func signIn(t *testing.T, ts *httptest.Server) auth.AuthResponse {
	t.Helper()
	body := `{"email":"test@example.com","password":"test-password","api":"004","code_verifier":"verifier"}`
	resp, err := http.Post(ts.URL+"/v1/login", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("login POST error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", resp.StatusCode)
	}
	var authResp auth.AuthResponse
	json.NewDecoder(resp.Body).Decode(&authResp)
	return authResp
}

func syncRequest(t *testing.T, ts *httptest.Server, accessToken, body string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("POST", ts.URL+"/v1/items", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sync POST error = %v", err)
	}
	return resp
}

// --- Integration Tests ---

func TestIntegration_RegisterSignInSync(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	// Register
	regResp := register(t, ts)
	if regResp.User.Email != "test@example.com" {
		t.Fatalf("email = %q", regResp.User.Email)
	}

	// Sign in
	signinResp := signIn(t, ts)
	if signinResp.Session.AccessToken == "" {
		t.Fatal("sign-in access token empty")
	}

	// Sync (empty first sync)
	resp := syncRequest(t, ts, signinResp.Session.AccessToken, `{"items":[]}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("sync status = %d", resp.StatusCode)
	}
}

func TestIntegration_RegisterGetParamsSignIn(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	register(t, ts)

	// Get params (POST with JSON body)
	paramsBody := `{"email":"test@example.com"}`
	resp, err := http.Post(ts.URL+"/v1/login-params", "application/json", bytes.NewBufferString(paramsBody))
	if err != nil {
		t.Fatalf("POST login-params error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("params status = %d", resp.StatusCode)
	}
	var kp map[string]any
	json.NewDecoder(resp.Body).Decode(&kp)
	if kp["identifier"] != "test@example.com" {
		t.Errorf("identifier = %v", kp["identifier"])
	}

	// Sign in
	signinResp := signIn(t, ts)
	if signinResp.Session.AccessToken == "" {
		t.Fatal("sign-in failed")
	}
}

func TestIntegration_SyncCreateRetrieveNote(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	regResp := register(t, ts)
	token := regResp.Session.AccessToken

	// Create a note
	createBody := `{"items":[{"uuid":"00000000-0000-0000-0000-000000000001","content":"encrypted-note","content_type":"Note"}]}`
	resp1 := syncRequest(t, ts, token, createBody)
	defer resp1.Body.Close()

	var syncResp1 map[string]any
	json.NewDecoder(resp1.Body).Decode(&syncResp1)
	syncToken := syncResp1["sync_token"].(string)
	savedItems := syncResp1["saved_items"].([]any)
	if len(savedItems) != 1 {
		t.Fatalf("saved %d items, want 1", len(savedItems))
	}

	// Retrieve with sync token (should get nothing new)
	retrieveBody, _ := json.Marshal(map[string]string{"sync_token": syncToken})
	resp2 := syncRequest(t, ts, token, string(retrieveBody))
	defer resp2.Body.Close()

	var syncResp2 map[string]any
	json.NewDecoder(resp2.Body).Decode(&syncResp2)
	retrieved := syncResp2["retrieved_items"].([]any)
	if len(retrieved) != 0 {
		t.Errorf("retrieved %d items on consecutive sync, want 0", len(retrieved))
	}

	// Full sync (no token) should return the note
	resp3 := syncRequest(t, ts, token, `{}`)
	defer resp3.Body.Close()

	var syncResp3 map[string]any
	json.NewDecoder(resp3.Body).Decode(&syncResp3)
	retrieved3 := syncResp3["retrieved_items"].([]any)
	if len(retrieved3) != 1 {
		t.Errorf("retrieved %d items on full sync, want 1", len(retrieved3))
	}
}

func TestIntegration_SyncUpdateConflict(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	regResp := register(t, ts)
	token := regResp.Session.AccessToken

	// Create a note
	createBody := `{"items":[{"uuid":"00000000-0000-0000-0000-000000000001","content":"v1","content_type":"Note"}]}`
	resp1 := syncRequest(t, ts, token, createBody)
	resp1.Body.Close()

	// Update the note
	updateBody := `{"items":[{"uuid":"00000000-0000-0000-0000-000000000001","content":"v2","content_type":"Note"}]}`
	resp2 := syncRequest(t, ts, token, updateBody)
	defer resp2.Body.Close()

	var syncResp map[string]any
	json.NewDecoder(resp2.Body).Decode(&syncResp)
	saved := syncResp["saved_items"].([]any)
	if len(saved) != 1 {
		t.Errorf("saved %d items on update, want 1", len(saved))
	}
}

func TestIntegration_SessionRefreshThenSync(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	regResp := register(t, ts)

	// Refresh session
	refreshBody, _ := json.Marshal(map[string]string{
		"access_token":  regResp.Session.AccessToken,
		"refresh_token": regResp.Session.RefreshToken,
	})
	refreshResp, _ := http.Post(ts.URL+"/v1/sessions/refresh", "application/json", bytes.NewBuffer(refreshBody))
	defer refreshResp.Body.Close()
	if refreshResp.StatusCode != http.StatusOK {
		t.Fatalf("refresh status = %d", refreshResp.StatusCode)
	}

	var newAuth auth.AuthResponse
	json.NewDecoder(refreshResp.Body).Decode(&newAuth)

	// Sync with new token
	resp := syncRequest(t, ts, newAuth.Session.AccessToken, `{}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("sync with refreshed token status = %d", resp.StatusCode)
	}
}

func TestIntegration_SignOutThenSync401(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	regResp := register(t, ts)
	token := regResp.Session.AccessToken

	// Sign out (POST /v1/logout)
	req, _ := http.NewRequest("POST", ts.URL+"/v1/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	signoutResp, _ := http.DefaultClient.Do(req)
	signoutResp.Body.Close()
	if signoutResp.StatusCode != http.StatusNoContent {
		t.Fatalf("signout status = %d, want 204", signoutResp.StatusCode)
	}

	// Try to sync - should be 401
	resp := syncRequest(t, ts, token, `{}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("sync after signout status = %d, want 401", resp.StatusCode)
	}
}

func TestIntegration_Meta(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/v1/meta")
	if err != nil {
		t.Fatalf("GET /v1/meta error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("meta status = %d", resp.StatusCode)
	}
	var meta map[string]any
	json.NewDecoder(resp.Body).Decode(&meta)
	if _, ok := meta["auth"]; !ok {
		t.Error("meta response missing 'auth' key")
	}
	if _, ok := meta["sync"]; !ok {
		t.Error("meta response missing 'sync' key")
	}
}

func TestIntegration_Healthcheck(t *testing.T) {
	ts := testServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthcheck")
	if err != nil {
		t.Fatalf("GET healthcheck error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthcheck status = %d", resp.StatusCode)
	}
}
