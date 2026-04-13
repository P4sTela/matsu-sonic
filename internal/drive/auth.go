package drive

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/P4sTela/matsu-sonic/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	driveapi "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Authenticate creates a Drive service using the configured auth method.
func Authenticate(ctx context.Context, cfg *config.Config) (*driveapi.Service, error) {
	switch cfg.AuthMethod {
	case "service_account":
		return authServiceAccount(ctx, cfg)
	default:
		return authOAuth(ctx, cfg)
	}
}

func authServiceAccount(ctx context.Context, cfg *config.Config) (*driveapi.Service, error) {
	keyJSON, err := os.ReadFile(cfg.CredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("read service account key: %w", err)
	}

	creds, err := google.CredentialsFromJSON(ctx, keyJSON, cfg.Scopes...)
	if err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}

	return driveapi.NewService(ctx, option.WithCredentials(creds))
}

func authOAuth(ctx context.Context, cfg *config.Config) (*driveapi.Service, error) {
	credJSON, err := os.ReadFile(cfg.CredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("read credentials: %w", err)
	}

	oauthCfg, err := google.ConfigFromJSON(credJSON, cfg.Scopes...)
	if err != nil {
		return nil, fmt.Errorf("parse oauth config: %w", err)
	}

	tok, err := loadToken(cfg.TokenPath)
	if err != nil {
		tok, err = obtainToken(ctx, oauthCfg)
		if err != nil {
			return nil, err
		}
		if err := saveToken(cfg.TokenPath, tok); err != nil {
			return nil, err
		}
	}

	client := oauthCfg.Client(ctx, tok)
	return driveapi.NewService(ctx, option.WithHTTPClient(client))
}

func loadToken(path string) (*oauth2.Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

func saveToken(path string, tok *oauth2.Token) error {
	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// obtainToken starts a local HTTP server to receive the OAuth callback.
func obtainToken(ctx context.Context, cfg *oauth2.Config) (*oauth2.Token, error) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	cfg.RedirectURL = fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			fmt.Fprintln(w, "Error: no authorization code received")
			return
		}
		codeCh <- code
		fmt.Fprintln(w, "Authorization successful! You can close this window.")
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(listener)
	defer srv.Shutdown(ctx)

	authURL := cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Open this URL in your browser:\n%s\n", authURL)

	select {
	case code := <-codeCh:
		return cfg.Exchange(ctx, code)
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
