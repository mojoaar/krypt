package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const gistAPIBase = "https://api.github.com/gists"
const gistFilename = "krypt.vault"

type gistFile struct {
	Content string `json:"content"`
}

type gistPayload struct {
	Description string              `json:"description"`
	Public      bool                `json:"public"`
	Files       map[string]gistFile `json:"files"`
}

type gistResponse struct {
	ID    string                 `json:"id"`
	Files map[string]gistGetFile `json:"files"`
}

type gistGetFile struct {
	Content string `json:"content"`
}

func resolveToken(cfg *Config) string {
	if t := os.Getenv("KRYPT_GITHUB_TOKEN"); t != "" {
		return t
	}
	return cfg.Token
}

// SyncPush uploads the encrypted vault blob as a GitHub Gist.
// If cfg.GistID is empty a new Gist is created and cfg.GistID is updated.
func SyncPush(cfg *Config, encryptedBlob []byte, masterPassword string) error {
	token := resolveToken(cfg)
	if token == "" {
		return fmt.Errorf("no GitHub token: set KRYPT_GITHUB_TOKEN or configure sync")
	}

	payload := gistPayload{
		Description: fmt.Sprintf("krypt vault — %s", time.Now().Format(time.RFC3339)),
		Public:      false,
		Files:       map[string]gistFile{gistFilename: {Content: string(encryptedBlob)}},
	}
	body, _ := json.Marshal(payload)

	var (
		req *http.Request
		err error
	)
	if cfg.GistID == "" {
		req, err = http.NewRequest(http.MethodPost, gistAPIBase, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(http.MethodPatch, gistAPIBase+"/"+cfg.GistID, bytes.NewReader(body))
	}
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github API %d: %s", resp.StatusCode, b)
	}

	var gr gistResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	cfg.GistID = gr.ID
	return nil
}

// SyncPull fetches the encrypted vault blob from the configured GitHub Gist.
func SyncPull(cfg *Config) ([]byte, error) {
	token := resolveToken(cfg)
	if token == "" {
		return nil, fmt.Errorf("no GitHub token: set KRYPT_GITHUB_TOKEN or configure sync")
	}
	if cfg.GistID == "" {
		return nil, fmt.Errorf("no Gist ID configured — push first")
	}

	req, err := http.NewRequest(http.MethodGet, gistAPIBase+"/"+cfg.GistID, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "token "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github API %d: %s", resp.StatusCode, b)
	}

	var gr gistResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	f, ok := gr.Files[gistFilename]
	if !ok {
		return nil, fmt.Errorf("vault file not found in gist")
	}
	return []byte(f.Content), nil
}
