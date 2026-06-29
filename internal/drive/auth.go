package drive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

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
		tok, err = reauthorize(ctx, oauthCfg, cfg.TokenPath)
		if err != nil {
			return nil, err
		}
	} else {
		// Validate the stored token. TokenSource refreshes it if the access
		// token is expired. If the refresh token itself is revoked/expired,
		// the token endpoint rejects it (invalid_grant) and we re-authenticate.
		refreshed, rerr := oauthCfg.TokenSource(ctx, tok).Token()
		switch {
		case rerr == nil:
			if refreshed.AccessToken != tok.AccessToken {
				tok = refreshed
				_ = saveToken(cfg.TokenPath, tok)
			}
		case isTokenRejected(rerr):
			log.Printf("[auth] stored token rejected (%v); re-authenticating", rerr)
			tok, err = reauthorize(ctx, oauthCfg, cfg.TokenPath)
			if err != nil {
				return nil, err
			}
		default:
			// Transient/network error: don't wipe the token or block on an
			// interactive flow — surface the error to the caller.
			return nil, fmt.Errorf("refresh oauth token: %w", rerr)
		}
	}

	client := oauthCfg.Client(ctx, tok)
	return driveapi.NewService(ctx, option.WithHTTPClient(client))
}

// reauthorize runs the interactive OAuth flow and persists the new token.
func reauthorize(ctx context.Context, oauthCfg *oauth2.Config, tokenPath string) (*oauth2.Token, error) {
	tok, err := obtainToken(ctx, oauthCfg)
	if err != nil {
		return nil, err
	}
	if err := saveToken(tokenPath, tok); err != nil {
		return nil, err
	}
	return tok, nil
}

// isTokenRejected reports whether the token endpoint rejected our refresh
// token (e.g. invalid_grant), as opposed to a transient/network failure.
func isTokenRejected(err error) bool {
	var re *oauth2.RetrieveError
	return errors.As(err, &re)
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

// AuthFlow represents an in-progress interactive OAuth authorization.
// The caller presents AuthURL to the user and waits on Done for completion.
type AuthFlow struct {
	AuthURL string
	Done    chan error

	srv *http.Server
}

// BeginOAuth starts an interactive OAuth flow for the configured credentials.
// It opens a local callback listener and returns the authorization URL to show
// the user. When the user approves, the token is exchanged, saved, and Done is
// signalled (nil on success). The flow auto-expires after 5 minutes.
func BeginOAuth(cfg *config.Config) (*AuthFlow, error) {
	if cfg.AuthMethod == "service_account" {
		return nil, fmt.Errorf("interactive authentication is not applicable for service account auth")
	}

	credJSON, err := os.ReadFile(cfg.CredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("read credentials: %w", err)
	}
	oauthCfg, err := google.ConfigFromJSON(credJSON, cfg.Scopes...)
	if err != nil {
		return nil, fmt.Errorf("parse oauth config: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	oauthCfg.RedirectURL = fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	flow := &AuthFlow{
		// ApprovalForce ensures Google returns a refresh token even on re-auth.
		AuthURL: oauthCfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce),
		Done:    make(chan error, 1),
	}

	tokenPath := cfg.TokenPath
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			fmt.Fprintln(w, "Authorization failed: no code received. You can close this window.")
			flow.finish(fmt.Errorf("no code in callback"))
			return
		}
		tok, err := oauthCfg.Exchange(context.Background(), code)
		if err != nil {
			fmt.Fprintln(w, "Authorization failed during token exchange. You can close this window.")
			flow.finish(fmt.Errorf("exchange: %w", err))
			return
		}
		if err := saveToken(tokenPath, tok); err != nil {
			fmt.Fprintln(w, "Authorization succeeded but saving the token failed. You can close this window.")
			flow.finish(fmt.Errorf("save token: %w", err))
			return
		}
		fmt.Fprintln(w, "Authorization successful! You can close this window and return to gdrive-sync.")
		flow.finish(nil)
	})

	flow.srv = &http.Server{Handler: mux}
	go flow.srv.Serve(listener)

	// Auto-expire so a never-completed flow does not leak the listener.
	time.AfterFunc(5*time.Minute, func() { flow.finish(fmt.Errorf("authorization timed out")) })

	return flow, nil
}

// finish signals completion exactly once and shuts down the callback server.
func (f *AuthFlow) finish(err error) {
	select {
	case f.Done <- err:
		if f.srv != nil {
			go f.srv.Shutdown(context.Background())
		}
	default:
		// Already finished.
	}
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
