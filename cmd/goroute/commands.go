package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/ziuus/goroute/pkg/catalog"
	"github.com/ziuus/goroute/pkg/config"
	"github.com/ziuus/goroute/pkg/providers"
	"github.com/ziuus/goroute/pkg/providers/antigravity"
	"github.com/ziuus/goroute/pkg/providers/codex"
	"github.com/ziuus/goroute/pkg/providers/copilot"
	"github.com/ziuus/goroute/pkg/providers/cursor"
	"github.com/ziuus/goroute/pkg/providers/kiro"
	"github.com/ziuus/goroute/pkg/providers/mistral"
	"github.com/ziuus/goroute/pkg/providers/openai"
	"github.com/ziuus/goroute/pkg/proxy"
	"github.com/ziuus/goroute/pkg/quota"
	"github.com/ziuus/goroute/pkg/router"
	"github.com/ziuus/goroute/pkg/store"
	"github.com/ziuus/goroute/pkg/sync"
	"github.com/spf13/cobra"
)

func init() {
	// Add subcommands to rootCmd in main
}

func defaultPaths() (configPath, storePath string) {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".goroute")
	return filepath.Join(baseDir, "config.yaml"), filepath.Join(baseDir, "vault.json")
}

func newAPIKeyProvider(pCfg config.ProviderConfig) providers.Provider {
	var keys []string
	for _, account := range pCfg.Accounts {
		if account.APIKey != "" {
			keys = append(keys, account.APIKey)
		}
	}
	if pCfg.APIKey != "" {
		keys = append(keys, pCfg.APIKey)
	}

	baseURL := pCfg.BaseURL
	if baseURL == "" {
		if entry := catalog.FindEntry(pCfg.ID); entry != nil && entry.BaseURL != "" {
			baseURL = entry.BaseURL
		} else if pCfg.ID == "agentrouter" {
			baseURL = "https://api.agentrouter.org/v1"
		} else if pCfg.ID == "xkiro" {
			baseURL = "https://api.xkiro.com/v1"
		} else if pCfg.ID == "openai" {
			baseURL = "https://api.openai.com/v1"
		}
	}

	var provider interface {
		providers.Provider
		SetAccountKey(string, string)
	}
	if pCfg.Type == "mistral" {
		provider = mistral.NewProviderWithKeys(pCfg.ID, baseURL, keys, pCfg.Headers)
	} else {
		provider = openai.NewProviderWithKeys(pCfg.ID, baseURL, keys, pCfg.Headers)
	}
	for _, account := range pCfg.Accounts {
		if account.APIKey != "" {
			provider.SetAccountKey(account.ID, account.APIKey)
		}
	}
	if pCfg.APIKey != "" {
		provider.SetAccountKey(pCfg.ID+"-primary", pCfg.APIKey)
	}
	return provider
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "linux":
		_ = exec.Command("xdg-open", url).Start()
	case "darwin":
		_ = exec.Command("open", url).Start()
	case "windows":
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
}

func NewServeCmd() *cobra.Command {
	var port int
	var cfgPath string
	var noDashboard bool

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the GoRoute gateway server",
		RunE: func(cmd *cobra.Command, args []string) error {
			defaultCfg, defaultStore := defaultPaths()
			if cfgPath == "" {
				cfgPath = defaultCfg
			}

			// Load config if exists, otherwise create default
			cfg, err := config.Load(cfgPath)
			if err != nil {
				fmt.Printf("Notice: No config at %s, using defaults.\n", cfgPath)
				cfg = config.DefaultConfig()
				_ = cfg.Save(cfgPath)
			}

			if port != 0 {
				cfg.Server.Port = port
			}

			st, _ := store.NewStore(defaultStore, "goroute-default-key")
			r := router.NewRouter()
			limiter := quota.NewLimiter()

			// Register configured providers
			for _, pCfg := range cfg.Providers {
				switch pCfg.Type {
				case "openai", "mistral":
					r.RegisterProvider(newAPIKeyProvider(pCfg))
				case "antigravity":
					var accounts []*antigravity.Account
					for _, a := range pCfg.Accounts {
						refToken := a.RefreshToken
						if refToken == "" && st != nil {
							sec, _ := st.GetSecret(a.ID)
							refToken = sec
						}
						accounts = append(accounts, &antigravity.Account{
							ID:           a.ID,
							Email:        a.Email,
							RefreshToken: refToken,
							AccessToken:  a.AccessToken,
							ExpiresAt:    a.ExpiresAt,
							ProjectID:    a.ProjectID,
						})
					}
					prov := antigravity.NewProvider(pCfg.ID, accounts)
					r.RegisterProvider(prov)
				case "kiro":
					var accounts []*kiro.Account
					for _, a := range pCfg.Accounts {
						refToken := a.RefreshToken
						if refToken == "" && st != nil {
							sec, _ := st.GetSecret(a.ID)
							refToken = sec
						}
						accounts = append(accounts, &kiro.Account{
							ID:           a.ID,
							RefreshToken: refToken,
							ProfileARN:   a.ProfileARN,
						})
					}
					if len(accounts) == 0 && pCfg.RefreshToken != "" {
						accounts = append(accounts, &kiro.Account{
							ID:           pCfg.ID,
							RefreshToken: pCfg.RefreshToken,
							ProfileARN:   pCfg.ProfileARN,
						})
					}
					prov := kiro.NewProviderWithAccounts(pCfg.ID, accounts)
					r.RegisterProvider(prov)
				case "codex":
					var accounts []*codex.Account
					for _, a := range pCfg.Accounts {
						refToken := a.RefreshToken
						if refToken == "" && st != nil {
							sec, _ := st.GetSecret(a.ID)
							refToken = sec
						}
						accounts = append(accounts, &codex.Account{
							ID:           a.ID,
							RefreshToken: refToken,
							AccessToken:  a.AccessToken,
							AccountID:    a.AccountID,
							ExpiresAt:    a.ExpiresAt,
						})
					}
					if len(accounts) == 0 && pCfg.RefreshToken != "" {
						accounts = append(accounts, &codex.Account{
							ID:           pCfg.ID,
							RefreshToken: pCfg.RefreshToken,
						})
					}
					prov := codex.NewProvider(pCfg.ID, accounts)
					r.RegisterProvider(prov)
				case "copilot":
					var accounts []*copilot.Account
					for _, a := range pCfg.Accounts {
						token := a.APIKey
						if token == "" {
							token = a.AccessToken
						}
						accounts = append(accounts, &copilot.Account{
							ID:          a.ID,
							GitHubToken: token,
							User:        a.Email,
						})
					}
					if len(accounts) == 0 && pCfg.APIKey != "" {
						accounts = append(accounts, &copilot.Account{
							ID:          pCfg.ID,
							GitHubToken: pCfg.APIKey,
						})
					}
					prov := copilot.NewProvider(pCfg.ID, accounts)
					r.RegisterProvider(prov)
				}
			}

			// Register configured combos
			for _, combo := range cfg.Combos {
				var targets []router.Target
				for _, t := range combo.Targets {
					targets = append(targets, router.Target{
						ProviderID: t.ProviderID,
						Model:      t.Model,
					})
				}
				r.RegisterCombo(combo.Name, combo.Strategy, targets)
			}

			addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

			if !noDashboard {
				if cfg.Server.DisableDashboard || os.Getenv("GOROUTE_NO_DASHBOARD") == "1" || os.Getenv("GOROUTE_NO_DASHBOARD") == "true" {
					noDashboard = true
				}
			}

			if noDashboard {
				fmt.Printf("⚡ GoRoute starting on http://%s (Headless mode, dashboard disabled)\n", addr)
			} else {
				fmt.Printf("⚡ GoRoute starting on http://%s\n", addr)
				fmt.Printf("   Dashboard available at:    http://%s/\n", addr)
			}
			fmt.Printf("   Universal OpenAI endpoint: http://%s/v1\n", addr)
			fmt.Printf("   Memory footprint:          < 15 MB RAM\n")

			srv := proxy.NewServer(r, limiter, !noDashboard)
			srv.SetConfigPaths(cfgPath, defaultStore)
			if err := srv.Reload(); err != nil {
				fmt.Printf("⚠️ Initial configuration load warning: %v\n", err)
			}

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGHUP)
			go func() {
				for range sigChan {
					fmt.Println("⚡ Received SIGHUP, reloading GoRoute configuration...")
					if err := srv.Reload(); err != nil {
						fmt.Printf("❌ Reload error: %v\n", err)
					}
				}
			}()

			return http.ListenAndServe(addr, srv.Handler())
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8128, "Port to listen on")
	cmd.Flags().StringVarP(&cfgPath, "config", "c", "", "Config file path")
	cmd.Flags().BoolVar(&noDashboard, "no-dashboard", false, "Disable the embedded web dashboard completely")
	return cmd
}

func NewAuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication and OAuth provider logins",
	}

	loginCmd := &cobra.Command{
		Use:   "login [provider]",
		Short: "Login to an OAuth provider (e.g. antigravity)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			providerName := args[0]
			if providerName != "antigravity" && providerName != "google" {
				return fmt.Errorf("currently supported OAuth providers: antigravity")
			}

			defaultCfg, defaultStore := defaultPaths()
			st, err := store.NewStore(defaultStore, "goroute-default-key")
			if err != nil {
				return err
			}

			verifier, challenge, err := antigravity.GeneratePKCE()
			if err != nil {
				return err
			}

			redirectURI := "http://127.0.0.1:20129/callback"
			state := "goroute-auth-" + fmt.Sprintf("%d", time.Now().UnixNano())
			authURL := antigravity.BuildAuthURL(redirectURI, state, challenge)

			codeChan := make(chan string, 1)
			errChan := make(chan error, 1)

			listener, err := net.Listen("tcp", "127.0.0.1:20129")
			if err != nil {
				return fmt.Errorf("failed to start local auth listener on port 20129: %w", err)
			}
			defer listener.Close()

			mux := http.NewServeMux()
			mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				if q.Get("state") != state {
					http.Error(w, "Invalid state", http.StatusBadRequest)
					errChan <- fmt.Errorf("OAuth state mismatch")
					return
				}
				code := q.Get("code")
				if code == "" {
					http.Error(w, "No code provided", http.StatusBadRequest)
					errChan <- fmt.Errorf("no authorization code in callback")
					return
				}

				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(`<h2>Authentication Successful!</h2><p>You can close this tab and return to your terminal.</p>`))
				codeChan <- code
			})

			server := &http.Server{Handler: mux}
			go func() {
				if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
					errChan <- err
				}
			}()

			fmt.Println("👉 Please open the following URL in your browser to sign in:")
			fmt.Println(authURL)
			fmt.Println("\nWaiting for browser callback on http://127.0.0.1:20129/callback...")
			openBrowser(authURL)

			var code string
			select {
			case code = <-codeChan:
			case err := <-errChan:
				return err
			case <-time.After(3 * time.Minute):
				return fmt.Errorf("OAuth login timed out after 3 minutes")
			}

			_ = server.Shutdown(context.Background())

			fmt.Println("⏳ Exchanging code for credentials & discovering project...")
			acc, err := antigravity.ExchangeCode(context.Background(), code, redirectURI, verifier)
			if err != nil {
				return fmt.Errorf("token exchange failed: %w", err)
			}

			accID := fmt.Sprintf("antigravity-%s", acc.Email)
			acc.ID = accID

			// Save to encrypted vault
			if acc.RefreshToken != "" {
				_ = st.SetSecret(accID, acc.RefreshToken)
			}

			// Update config.yaml
			cfg, _ := config.Load(defaultCfg)
			if cfg == nil {
				cfg = config.DefaultConfig()
			}

			// Find or create antigravity provider
			found := false
			for i, p := range cfg.Providers {
				if p.Type == "antigravity" {
					cfg.Providers[i].Accounts = append(cfg.Providers[i].Accounts, config.AccountConfig{
						ID:          accID,
						Email:       acc.Email,
						ProjectID:   acc.ProjectID,
						AccessToken: acc.AccessToken,
						ExpiresAt:   acc.ExpiresAt,
					})
					found = true
					break
				}
			}
			if !found {
				cfg.Providers = append(cfg.Providers, config.ProviderConfig{
					ID:   "antigravity",
					Type: "antigravity",
					Accounts: []config.AccountConfig{
						{
							ID:          accID,
							Email:       acc.Email,
							ProjectID:   acc.ProjectID,
							AccessToken: acc.AccessToken,
							ExpiresAt:   acc.ExpiresAt,
						},
					},
				})
			}

			_ = cfg.Save(defaultCfg)

			fmt.Println("✅ Successfully connected Antigravity account!")
			fmt.Printf("   Account ID: %s\n", acc.ID)
			fmt.Printf("   Email:      %s\n", acc.Email)
			fmt.Printf("   Project ID: %s\n", acc.ProjectID)
			return nil
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List configured providers and accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			defaultCfg, _ := defaultPaths()
			cfg, err := config.Load(defaultCfg)
			if err != nil {
				fmt.Println("No configuration found. Run 'goroute serve' or 'goroute auth login antigravity'")
				return nil
			}

			fmt.Println("Configured Providers:")
			for _, p := range cfg.Providers {
				fmt.Printf("• Provider: %s (Type: %s)\n", p.ID, p.Type)
				for _, acc := range p.Accounts {
					fmt.Printf("   - Account: %s (%s) [Project: %s]\n", acc.ID, acc.Email, acc.ProjectID)
				}
			}

			if len(cfg.Combos) > 0 {
				fmt.Println("\nConfigured Combos:")
				for _, c := range cfg.Combos {
					fmt.Printf("• Combo: %s [Strategy: %s]\n", c.Name, c.Strategy)
					for _, t := range c.Targets {
						fmt.Printf("   -> %s / %s\n", t.ProviderID, t.Model)
					}
				}
			}
			return nil
		},
	}

	importCmd := &cobra.Command{
		Use:   "import [provider]",
		Short: "Auto-import credentials from local tools (e.g. kiro)",
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "all"
			if len(args) > 0 {
				target = args[0]
			}

			defaultCfg, _ := defaultPaths()
			cfg, err := config.Load(defaultCfg)
			if err != nil {
				cfg = config.DefaultConfig()
			}

			imported := 0

			// 0. Import Antigravity (Local IDE & CLI)
			if target == "antigravity" || target == "all" {
				accounts, err := antigravity.AutoImportLocalAntigravity(cmd.Context())
				if err == nil && len(accounts) > 0 {
					pIdx := -1
					for i, p := range cfg.Providers {
						if p.ID == "antigravity" {
							pIdx = i
							break
						}
					}
					if pIdx == -1 {
						cfg.Providers = append(cfg.Providers, config.ProviderConfig{
							ID:       "antigravity",
							Type:     "antigravity",
							Strategy: "priority",
						})
						pIdx = len(cfg.Providers) - 1
					}

					for _, acc := range accounts {
						foundAcc := false
						for j, existing := range cfg.Providers[pIdx].Accounts {
							if existing.RefreshToken == acc.RefreshToken {
								cfg.Providers[pIdx].Accounts[j].AccessToken = acc.AccessToken
								if acc.Email != "" {
									cfg.Providers[pIdx].Accounts[j].Email = acc.Email
								}
								if acc.ProjectID != "" {
									cfg.Providers[pIdx].Accounts[j].AccountID = acc.ProjectID
								}
								foundAcc = true
								break
							}
						}
						if !foundAcc {
							accID := fmt.Sprintf("agy-acc-%d", len(cfg.Providers[pIdx].Accounts)+1)
							cfg.Providers[pIdx].Accounts = append(cfg.Providers[pIdx].Accounts, config.AccountConfig{
								ID:           accID,
								Pool:         "pool1",
								RefreshToken: acc.RefreshToken,
								AccessToken:  acc.AccessToken,
								Email:        acc.Email,
								AccountID:    acc.ProjectID,
								Weight:       1,
							})
							imported++
							fmt.Printf("✅ Imported local Antigravity account: %s (%s, project: %s)\n",
								accID, acc.Email, acc.ProjectID)
						}
					}
					if len(cfg.Providers[pIdx].Accounts) > 0 && cfg.Providers[pIdx].RefreshToken == "" {
						cfg.Providers[pIdx].RefreshToken = cfg.Providers[pIdx].Accounts[0].RefreshToken
					}
				} else if target == "antigravity" {
					return fmt.Errorf("failed to import local antigravity accounts: %v", err)
				}
			}

			// 1. Import Kiro
			if target == "kiro" || target == "all" {
				home, _ := os.UserHomeDir()
				dbPath := filepath.Join(home, ".local", "share", "kiro-cli", "data.sqlite3")
				if _, err := os.Stat(dbPath); err == nil {
					out, err := exec.Command("sqlite3", dbPath, "SELECT value FROM auth_kv WHERE key='kirocli:social:token';").Output()
					if err == nil && len(out) > 0 {
						var tokData struct {
							RefreshToken string `json:"refresh_token"`
							ProfileArn   string `json:"profile_arn"`
						}
						if json.Unmarshal(out, &tokData) == nil && tokData.RefreshToken != "" {
							if tokData.ProfileArn == "" {
								tokData.ProfileArn = "arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK"
							}
							found := false
							for i, p := range cfg.Providers {
								if p.ID == "kiro" {
									found = true
									updated := false
									for j, a := range cfg.Providers[i].Accounts {
										if strings.HasPrefix(a.ID, "kiro-cli") {
											cfg.Providers[i].Accounts[j].RefreshToken = tokData.RefreshToken
											cfg.Providers[i].Accounts[j].ProfileARN = tokData.ProfileArn
											updated = true
											break
										}
									}
									if !updated {
										cfg.Providers[i].Accounts = append(cfg.Providers[i].Accounts, config.AccountConfig{
											ID:           "kiro-cli",
											RefreshToken: tokData.RefreshToken,
											ProfileARN:   tokData.ProfileArn,
										})
									}
									break
								}
							}
							if !found {
								cfg.Providers = append(cfg.Providers, config.ProviderConfig{
									ID:           "kiro",
									Type:         "kiro",
									RefreshToken: tokData.RefreshToken,
									ProfileARN:   tokData.ProfileArn,
									Accounts: []config.AccountConfig{
										{
											ID:           "kiro-cli",
											RefreshToken: tokData.RefreshToken,
											ProfileARN:   tokData.ProfileArn,
										},
									},
								})
							}
							fmt.Println("✅ Successfully imported Kiro credentials from ~/.local/share/kiro-cli/data.sqlite3")
							imported++
						}
					}
				}

				if imported == 0 {
					// Fallback: Probe ~/.aws/sso/cache/*.json
					ssoDir := filepath.Join(home, ".aws", "sso", "cache")
					if entries, err := os.ReadDir(ssoDir); err == nil {
						for _, ent := range entries {
							if !ent.IsDir() && strings.HasSuffix(ent.Name(), ".json") {
								content, err := os.ReadFile(filepath.Join(ssoDir, ent.Name()))
								if err != nil {
									continue
								}
								var ssoData struct {
									RefreshToken string `json:"refreshToken"`
									AccessToken  string `json:"accessToken"`
									ProfileArn   string `json:"profileArn"`
								}
								if json.Unmarshal(content, &ssoData) == nil && (ssoData.RefreshToken != "" || ssoData.AccessToken != "") {
									token := ssoData.RefreshToken
									if token == "" {
										token = ssoData.AccessToken
									}
									arn := ssoData.ProfileArn
									if arn == "" {
										arn = "arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK"
									}
									found := false
									for i, p := range cfg.Providers {
										if p.ID == "kiro" {
											found = true
											cfg.Providers[i].Accounts = append(cfg.Providers[i].Accounts, config.AccountConfig{
												ID:           "kiro-sso",
												RefreshToken: token,
												ProfileARN:   arn,
											})
											break
										}
									}
									if !found {
										cfg.Providers = append(cfg.Providers, config.ProviderConfig{
											ID:           "kiro",
											Type:         "kiro",
											RefreshToken: token,
											ProfileARN:   arn,
											Accounts: []config.AccountConfig{
												{
													ID:           "kiro-sso",
													RefreshToken: token,
													ProfileARN:   arn,
												},
											},
										})
									}
									fmt.Printf("✅ Successfully imported Kiro credentials from ~/.aws/sso/cache/%s\n", ent.Name())
									imported++
									break
								}
							}
						}
					}
				}
				if target == "kiro" && imported == 0 {
					return fmt.Errorf("no Kiro credentials found in kiro-cli or ~/.aws/sso/cache")
				}
			}

			// 2. Import Codex
			if target == "codex" || target == "all" {
				acc, err := codex.ReadCodexCLIAuth()
				if err == nil && acc != nil {
					found := false
					for i, p := range cfg.Providers {
						if p.ID == "codex" {
							found = true
							updated := false
							for j, a := range cfg.Providers[i].Accounts {
								if a.ID == "codex-cli" {
									cfg.Providers[i].Accounts[j].RefreshToken = acc.RefreshToken
									cfg.Providers[i].Accounts[j].AccessToken = acc.AccessToken
									cfg.Providers[i].Accounts[j].AccountID = acc.AccountID
									updated = true
									break
								}
							}
							if !updated {
								cfg.Providers[i].Accounts = append(cfg.Providers[i].Accounts, config.AccountConfig{
									ID:           "codex-cli",
									RefreshToken: acc.RefreshToken,
									AccessToken:  acc.AccessToken,
									AccountID:    acc.AccountID,
								})
							}
							break
						}
					}
					if !found {
						cfg.Providers = append(cfg.Providers, config.ProviderConfig{
							ID:           "codex",
							Type:         "codex",
							RefreshToken: acc.RefreshToken,
							Accounts: []config.AccountConfig{
								{
									ID:           "codex-cli",
									RefreshToken: acc.RefreshToken,
									AccessToken:  acc.AccessToken,
									AccountID:    acc.AccountID,
								},
							},
						})
					}
					fmt.Println("✅ Successfully imported Codex credentials from ~/.codex/auth.json")
					imported++
				} else if target == "codex" {
					return fmt.Errorf("failed to import codex credentials: %v", err)
				}
			}

			// 3. Import Copilot
			if target == "copilot" || target == "all" {
				acc, err := copilot.ReadCopilotCLIAuth()
				if err == nil && acc != nil {
					found := false
					for i, p := range cfg.Providers {
						if p.ID == "copilot" {
							found = true
							updated := false
							for j, a := range cfg.Providers[i].Accounts {
								if a.ID == acc.ID {
									cfg.Providers[i].Accounts[j].APIKey = acc.GitHubToken
									cfg.Providers[i].Accounts[j].Email = acc.User
									updated = true
									break
								}
							}
							if !updated {
								cfg.Providers[i].Accounts = append(cfg.Providers[i].Accounts, config.AccountConfig{
									ID:     acc.ID,
									APIKey: acc.GitHubToken,
									Email:  acc.User,
								})
							}
							break
						}
					}
					if !found {
						cfg.Providers = append(cfg.Providers, config.ProviderConfig{
							ID:     "copilot",
							Type:   "copilot",
							APIKey: acc.GitHubToken,
							Accounts: []config.AccountConfig{
								{
									ID:     acc.ID,
									APIKey: acc.GitHubToken,
									Email:  acc.User,
								},
							},
						})
					}
					fmt.Println("✅ Successfully imported GitHub Copilot credentials from ~/.config/github-copilot")
					imported++
				} else if target == "copilot" {
					return fmt.Errorf("failed to import copilot credentials: %v", err)
				}
			}

			if target == "all" || target == "cursor" {
				acc, err := cursor.AutoImportCursor()
				if err == nil && acc != nil {
					found := false
					for i, p := range cfg.Providers {
						if p.ID == "cursor" {
							found = true
							accFound := false
							for j, a := range p.Accounts {
								if a.ID == acc.ID || a.Email == acc.Email {
									cfg.Providers[i].Accounts[j].AccessToken = acc.AccessToken
									cfg.Providers[i].Accounts[j].RefreshToken = acc.RefreshToken
									cfg.Providers[i].Accounts[j].ExpiresAt = acc.ExpiresAt
									accFound = true
									break
								}
							}
							if !accFound {
								cfg.Providers[i].Accounts = append(cfg.Providers[i].Accounts, config.AccountConfig{
									ID:           acc.ID,
									Email:        acc.Email,
									AccessToken:  acc.AccessToken,
									RefreshToken: acc.RefreshToken,
									ExpiresAt:    acc.ExpiresAt,
								})
							}
							break
						}
					}
					if !found {
						cfg.Providers = append(cfg.Providers, config.ProviderConfig{
							ID:   "cursor",
							Type: "cursor",
							Accounts: []config.AccountConfig{
								{
									ID:           acc.ID,
									Email:        acc.Email,
									AccessToken:  acc.AccessToken,
									RefreshToken: acc.RefreshToken,
									ExpiresAt:    acc.ExpiresAt,
								},
							},
						})
					}
					fmt.Println("✅ Successfully imported Cursor credentials from ~/.config/cursor/auth.json")
					imported++
				} else if target == "cursor" {
					return fmt.Errorf("failed to import cursor credentials: %v", err)
				}
			}

			// 5. Import OmniRoute Vault
			if target == "omniroute" || target == "all" {
				res, err := sync.SyncFromOmniRoute(defaultCfg)
				if err == nil && res != nil {
					fmt.Printf("✅ Successfully synced from OmniRoute: %d providers, %d accounts, %d combos\n",
						res.ProvidersImported, res.AccountsImported, res.CombosImported)
					imported += res.ProvidersImported + res.AccountsImported
				} else if target == "omniroute" {
					return fmt.Errorf("failed to import omniroute credentials: %v", err)
				}
			}

			if imported == 0 {
				return fmt.Errorf("no credentials found to import for %s", target)
			}

			if err := cfg.Save(defaultCfg); err != nil {
				return err
			}
			_, _ = http.Post("http://127.0.0.1:8128/api/reload", "application/json", nil)
			return nil
		},
	}

	authCmd.AddCommand(loginCmd, listCmd, importCmd)
	return authCmd
}

func NewStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check running GoRoute gateway status",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := http.Get("http://127.0.0.1:8128/health")
			if err != nil {
				fmt.Println("❌ GoRoute server is NOT running on port 8128.")
				fmt.Println("   Start it with: goroute serve")
				return nil
			}
			defer resp.Body.Close()

			var healthData struct {
				Status         string                 `json:"status"`
				Engine         string                 `json:"engine"`
				RustSIMD       bool                   `json:"rust_simd"`
				MemoryAlloc    string                 `json:"memory_alloc"`
				ProvidersCount int                    `json:"providers_count"`
				CombosCount    int                    `json:"combos_count"`
				Providers      map[string]interface{} `json:"providers"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&healthData)

			fmt.Println("✅ GoRoute is ACTIVE on http://127.0.0.1:8128")
			fmt.Printf("   Engine:          %s\n", healthData.Engine)
			fmt.Printf("   Memory RSS:      %s\n", healthData.MemoryAlloc)
			fmt.Printf("   Rust SIMD:       %v\n", healthData.RustSIMD)
			fmt.Printf("   Active Providers: %d\n", healthData.ProvidersCount)
			fmt.Printf("   Ready Combos:     %d\n", healthData.CombosCount)

			modelsResp, err := http.Get("http://127.0.0.1:8128/v1/models")
			if err == nil {
				defer modelsResp.Body.Close()
				var data struct {
					Data []providers.ModelInfo `json:"data"`
				}
				if json.NewDecoder(modelsResp.Body).Decode(&data) == nil {
					fmt.Printf("   Available Models/Combos: %d\n", len(data.Data))
					for _, m := range data.Data {
						fmt.Printf("   • %-26s [%s]\n", m.ID, m.OwnedBy)
					}
				}
			}
			return nil
		},
	}
}

func NewProviderCmd() *cobra.Command {
	providerCmd := &cobra.Command{
		Use:   "provider",
		Short: "Manage custom providers and endpoints",
	}

	var pType, baseURL, apiKey string
	addCmd := &cobra.Command{
		Use:   "add [id]",
		Short: "Add a custom endpoint or API key provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			defaultCfg, _ := defaultPaths()
			cfg, err := config.Load(defaultCfg)
			if err != nil {
				cfg = config.DefaultConfig()
			}

			if baseURL == "" {
				if entry := catalog.FindEntry(id); entry != nil {
					baseURL = entry.BaseURL
					pType = entry.Type
					fmt.Printf("ℹ Using catalog preset for '%s' (%s)\n", entry.Name, entry.BaseURL)
				} else {
					return fmt.Errorf("provider '%s' is not a known preset. Please specify --url", id)
				}
			}

			found := false
			for i, p := range cfg.Providers {
				if p.ID == id {
					cfg.Providers[i].BaseURL = baseURL
					cfg.Providers[i].APIKey = apiKey
					cfg.Providers[i].Type = pType
					found = true
					break
				}
			}
			if !found {
				cfg.Providers = append(cfg.Providers, config.ProviderConfig{
					ID:      id,
					Type:    pType,
					BaseURL: baseURL,
					APIKey:  apiKey,
				})
			}

			if err := cfg.Save(defaultCfg); err != nil {
				return err
			}

			fmt.Printf("✅ Provider '%s' saved successfully!\n", id)
			fmt.Printf("   Type:     %s\n", pType)
			fmt.Printf("   Base URL: %s\n", baseURL)
			return nil
		},
	}

	addCmd.Flags().StringVar(&pType, "type", "openai", "Provider type (openai, antigravity)")
	addCmd.Flags().StringVar(&baseURL, "url", "", "Custom Base URL endpoint (optional if using a catalog preset)")
	addCmd.Flags().StringVar(&apiKey, "key", "", "API Key for authentication")

	providerCmd.AddCommand(addCmd)
	return providerCmd
}

func NewCatalogCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "catalog [search]",
		Short: "Browse built-in provider presets (NVIDIA, Groq, DeepSeek, Ollama, etc.)",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) > 0 {
				query = args[0]
			}
			entries := catalog.Search(query)
			if len(entries) == 0 {
				fmt.Printf("No providers found matching '%s'\n", query)
				return nil
			}

			fmt.Println("Available Provider Presets in GoRoute:")
			fmt.Printf("%-14s %-25s %-8s %s\n", "ID", "NAME", "AUTH", "DEFAULT ENDPOINT")
			fmt.Println(strings.Repeat("-", 80))
			for _, e := range entries {
				fmt.Printf("%-14s %-25s %-8s %s\n", e.ID, e.Name, e.AuthType, e.BaseURL)
			}
			fmt.Println("\nTo add any preset in one step:")
			fmt.Println("  goroute provider add <id> --key <your-api-key>")
			return nil
		},
	}
}

func NewComboCmd() *cobra.Command {
	comboCmd := &cobra.Command{
		Use:   "combo",
		Short: "Manage smart fallback combos and cascades",
	}

	var strategy, targetsStr string
	addCmd := &cobra.Command{
		Use:   "add [name]",
		Short: "Create or update a failover combo",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			defaultCfg, _ := defaultPaths()
			cfg, err := config.Load(defaultCfg)
			if err != nil {
				cfg = config.DefaultConfig()
			}

			targetList := strings.Split(targetsStr, ",")
			var targets []config.ComboTarget
			for _, t := range targetList {
				parts := strings.Split(strings.TrimSpace(t), "/")
				if len(parts) == 2 {
					targets = append(targets, config.ComboTarget{
						ProviderID: parts[0],
						Model:      parts[1],
					})
				}
			}

			if len(targets) == 0 {
				return fmt.Errorf("invalid targets. Example format: --targets 'antigravity/claude-3-7-sonnet,tokenrouter/claude-3-7-sonnet'")
			}

			found := false
			for i, c := range cfg.Combos {
				if c.Name == name {
					cfg.Combos[i].Strategy = strategy
					cfg.Combos[i].Targets = targets
					found = true
					break
				}
			}
			if !found {
				cfg.Combos = append(cfg.Combos, config.ComboConfig{
					Name:     name,
					Strategy: strategy,
					Targets:  targets,
				})
			}

			if err := cfg.Save(defaultCfg); err != nil {
				return err
			}

			fmt.Printf("✅ Combo '%s' saved successfully!\n", name)
			fmt.Printf("   Strategy: %s\n", strategy)
			for _, t := range targets {
				fmt.Printf("   -> %s / %s\n", t.ProviderID, t.Model)
			}
			return nil
		},
	}

	addCmd.Flags().StringVar(&strategy, "strategy", "failover", "Routing strategy (failover, round_robin)")
	addCmd.Flags().StringVar(&targetsStr, "targets", "", "Comma-separated list of provider/model pairs")
	_ = addCmd.MarkFlagRequired("targets")

	comboCmd.AddCommand(addCmd)
	return comboCmd
}

func NewModelsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "models",
		Short: "List all active models and combos available in GoRoute",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := http.Get("http://127.0.0.1:8128/v1/models")
			if err != nil {
				defaultCfg, _ := defaultPaths()
				cfg, err := config.Load(defaultCfg)
				if err != nil {
					return fmt.Errorf("gateway not running and no config found")
				}
				fmt.Println("Available Combos (Configured):")
				for _, c := range cfg.Combos {
					fmt.Printf("• %s (Combo / %s)\n", c.Name, c.Strategy)
				}
				return nil
			}
			defer resp.Body.Close()

			var data struct {
				Data []providers.ModelInfo `json:"data"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return err
			}

			fmt.Println("Active Models & Combos:")
			for _, m := range data.Data {
				fmt.Printf("• %-30s [%s]\n", m.ID, m.OwnedBy)
			}
			return nil
		},
	}
}

func NewSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync [target]",
		Short: "Synchronize accounts, models, and combos from OmniRoute or external vaults",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			source := "omniroute"
			if len(args) > 0 {
				source = args[0]
			}
			defaultCfg, _ := defaultPaths()
			if source == "omniroute" {
				fmt.Println("⚡ Syncing providers, credentials, and combos from OmniRoute SQLite vault...")
				res, err := sync.SyncFromOmniRoute(defaultCfg)
				if err != nil {
					return fmt.Errorf("sync error: %w", err)
				}
				fmt.Printf("✅ Sync complete: %d providers, %d accounts, %d combos\n",
					res.ProvidersImported, res.AccountsImported, res.CombosImported)
				for _, d := range res.Details {
					fmt.Printf("   • %s\n", d)
				}
				// Signal running server to reload if active
				_, _ = http.Post("http://127.0.0.1:8128/api/reload", "application/json", nil)
				return nil
			}

			if source == "antigravity" {
				fmt.Println("⚡ Scanning and importing local Antigravity accounts from CLI & IDE...")
				accounts, err := antigravity.AutoImportLocalAntigravity(cmd.Context())
				if err != nil {
					return fmt.Errorf("antigravity import error: %w", err)
				}
				cfg, err := config.Load(defaultCfg)
				if err != nil {
					cfg = config.DefaultConfig()
				}
				pIdx := -1
				for i, p := range cfg.Providers {
					if p.ID == "antigravity" {
						pIdx = i
						break
					}
				}
				if pIdx == -1 {
					cfg.Providers = append(cfg.Providers, config.ProviderConfig{
						ID:       "antigravity",
						Type:     "antigravity",
						Strategy: "priority",
					})
					pIdx = len(cfg.Providers) - 1
				}
				imported := 0
				for _, acc := range accounts {
					foundAcc := false
					for j, existing := range cfg.Providers[pIdx].Accounts {
						if existing.RefreshToken == acc.RefreshToken {
							cfg.Providers[pIdx].Accounts[j].AccessToken = acc.AccessToken
							if acc.Email != "" {
								cfg.Providers[pIdx].Accounts[j].Email = acc.Email
							}
							foundAcc = true
							break
						}
					}
					if !foundAcc {
						accID := fmt.Sprintf("agy-acc-%d", len(cfg.Providers[pIdx].Accounts)+1)
						cfg.Providers[pIdx].Accounts = append(cfg.Providers[pIdx].Accounts, config.AccountConfig{
							ID:           accID,
							Pool:         "pool1",
							RefreshToken: acc.RefreshToken,
							AccessToken:  acc.AccessToken,
							Email:        acc.Email,
							AccountID:    acc.ProjectID,
							Weight:       1,
						})
						imported++
						fmt.Printf("✅ Imported local Antigravity account: %s (%s, project: %s)\n",
							accID, acc.Email, acc.ProjectID)
					}
				}
				if imported > 0 {
					if err := cfg.Save(defaultCfg); err != nil {
						return err
					}
					fmt.Printf("🎉 Successfully added %d new Antigravity account(s) to pool!\n", imported)
				} else {
					fmt.Println("ℹ All detected local Antigravity accounts are already in your pool.")
				}
				_, _ = http.Post("http://127.0.0.1:8128/api/reload", "application/json", nil)
				return nil
			}

			return fmt.Errorf("unknown sync target: %s (supported: omniroute, antigravity)", source)
		},
	}
}
