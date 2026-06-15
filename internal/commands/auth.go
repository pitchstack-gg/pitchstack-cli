package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/pitchstack-gg/pitchstack-cli/internal/pitchstack"

	"github.com/urfave/cli/v3"
)

func newLoginCommand() *cli.Command {
	return &cli.Command{
		Name:  "login",
		Usage: "Login (OAuth via browser by default)",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "legacy", Usage: "Use legacy email/password login instead of OAuth"},
			&cli.BoolFlag{Name: "no-open", Usage: "Don't attempt to open a browser (prints URL instead)"},
			&cli.DurationFlag{Name: "timeout", Usage: "Max time to wait for browser login to complete", Value: 5 * time.Minute},
			&cli.StringFlag{Name: "oauth-base-url", Usage: "OAuth web base URL (or set PITCHSTACK_OAUTH_BASE_URL)"},
			&cli.StringFlag{Name: "email", Usage: "Account email (legacy login only)"},
			&cli.StringFlag{Name: "device", Usage: "Optional device info label (legacy login only)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			useLegacy := cmd.Bool("legacy") || cmd.IsSet("email")
			if !useLegacy {
				oauthBaseURL := strings.TrimSpace(cmd.String("oauth-base-url"))
				if oauthBaseURL == "" {
					oauthBaseURL = strings.TrimSpace(os.Getenv("PITCHSTACK_OAUTH_BASE_URL"))
				}
				if oauthBaseURL == "" {
					oauthBaseURL = strings.TrimSpace(st.Profile.OAuthBaseURL)
				}
				if oauthBaseURL == "" {
					return cli.Exit("missing oauth base url; set profile.oauthBaseUrl or PITCHSTACK_OAUTH_BASE_URL", 2)
				}

				timeout := cmd.Duration("timeout")
				if timeout <= 0 {
					return cli.Exit("--timeout must be > 0", 2)
				}

				sess, err := st.Service.CreateCLILoginSession(ctx, oauthBaseURL)
				if err != nil {
					return err
				}

				if !cmd.Bool("no-open") {
					if err := openBrowser(sess.VerificationURL); err != nil {
						_, _ = fmt.Fprintf(cmd.ErrWriter, "failed to open browser: %s\n", err.Error())
						_, _ = fmt.Fprintln(cmd.ErrWriter, "Complete login in your browser:")
						_, _ = fmt.Fprintln(cmd.ErrWriter, sess.VerificationURL)
					}
				} else {
					_, _ = fmt.Fprintln(cmd.ErrWriter, "Complete login in your browser:")
					_, _ = fmt.Fprintln(cmd.ErrWriter, sess.VerificationURL)
				}

				pollCtx, cancel := context.WithTimeout(ctx, timeout)
				defer cancel()

				for {
					select {
					case <-pollCtx.Done():
						cancelCtx, cancelReq := context.WithTimeout(context.Background(), 5*time.Second)
						_ = st.Service.CancelCLILoginSession(cancelCtx, sess.SessionID, sess.SessionSecret)
						cancelReq()
						if errors.Is(pollCtx.Err(), context.DeadlineExceeded) {
							return cli.Exit("timed out waiting for login to complete; re-run with a longer --timeout", 1)
						}
						return pollCtx.Err()
					default:
					}

					out, err := st.Service.PollCLILoginSession(pollCtx, sess.SessionID, sess.SessionSecret)
					if err != nil {
						return err
					}

					switch strings.TrimSpace(out.Status) {
					case "CLI_LOGIN_SESSION_STATUS_COMPLETE":
						if out.Login == nil {
							return errors.New("login complete but missing token response")
						}
						sess, err := st.Service.SaveLoginResult(ctx, out.Login)
						if err != nil {
							return err
						}
						if strings.TrimSpace(sess.Username) != "" {
							_, err = fmt.Fprintf(cmd.Writer, "logged in as %s (%s)\n", sess.Username, sess.UserID)
							return err
						}
						_, err = fmt.Fprintf(cmd.Writer, "logged in (%s)\n", sess.UserID)
						return err
					case "CLI_LOGIN_SESSION_STATUS_EXPIRED":
						return cli.Exit("login session expired; please re-run login", 1)
					case "CLI_LOGIN_SESSION_STATUS_CANCELED":
						return cli.Exit("login canceled", 1)
					default:
						interval := sess.PollInterval
						if interval <= 0 {
							interval = 2 * time.Second
						}
						t := time.NewTimer(interval)
						select {
						case <-pollCtx.Done():
							t.Stop()
							continue
						case <-t.C:
						}
					}
				}
			}

			email := strings.TrimSpace(cmd.String("email"))
			password := ""
			device := strings.TrimSpace(cmd.String("device"))

			if email == "" {
				email, err = readPrompt(cmd, "Email: ")
				if err != nil {
					return err
				}
			}
			password, err = readSecret(cmd, "Password: ")
			if err != nil {
				return err
			}
			if email == "" || password == "" {
				return cli.Exit("email and password are required", 2)
			}

			sess, err := st.Service.Login(ctx, pitchstack.LoginInput{
				Email:      email,
				Password:   password,
				DeviceInfo: device,
			})
			if err != nil {
				return err
			}

			if strings.TrimSpace(sess.Username) != "" {
				_, err = fmt.Fprintf(cmd.Writer, "logged in as %s (%s)\n", sess.Username, sess.UserID)
				return err
			}
			_, err = fmt.Fprintf(cmd.Writer, "logged in (%s)\n", sess.UserID)
			return err
		},
	}
}

func openBrowser(url string) error {
	if strings.TrimSpace(url) == "" {
		return errors.New("empty url")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func newSignupCommand() *cli.Command {
	return &cli.Command{
		Name:  "signup",
		Usage: "Create a new account",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "email", Usage: "Account email"},
			&cli.StringFlag{Name: "username", Usage: "Optional username"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			email := strings.TrimSpace(cmd.String("email"))
			username := strings.TrimSpace(cmd.String("username"))
			password := ""

			if email == "" {
				email, err = readPrompt(cmd, "Email: ")
				if err != nil {
					return err
				}
			}
			if username == "" {
				username, err = readPrompt(cmd, "Username (optional): ")
				if err != nil {
					return err
				}
			}
			password, err = readSecret(cmd, "Password: ")
			if err != nil {
				return err
			}
			if email == "" || password == "" {
				return cli.Exit("email and password are required", 2)
			}

			sess, err := st.Service.Signup(ctx, pitchstack.SignupInput{
				Email:    email,
				Username: username,
				Password: password,
			})
			if err != nil {
				return err
			}

			if strings.TrimSpace(sess.Username) != "" {
				_, err = fmt.Fprintf(cmd.Writer, "account created; logged in as %s (%s)\n", sess.Username, sess.UserID)
				return err
			}
			_, err = fmt.Fprintf(cmd.Writer, "account created; logged in (%s)\n", sess.UserID)
			return err
		},
	}
}

func newWhoamiCommand() *cli.Command {
	return &cli.Command{
		Name:  "whoami",
		Usage: "Show current authenticated user",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			sess, err := st.Sessions.Load()
			if err != nil {
				return err
			}
			if sess == nil {
				_, err := fmt.Fprintln(cmd.Writer, "not logged in")
				return err
			}

			username := strings.TrimSpace(sess.Username)
			if username == "" {
				if u, err := st.Service.EnsureUsername(ctx); err == nil {
					username = strings.TrimSpace(u)
				}
			}

			me, err := st.Service.Me(ctx)
			if err != nil {
				return err
			}
			if me == nil || me.User == nil {
				_, err := fmt.Fprintln(cmd.Writer, "not logged in")
				return err
			}

			if username == "" {
				username = "unknown"
			}
			_, err = fmt.Fprintf(cmd.Writer, "%s (%s) <%s>\n", username, me.User.UserID, me.User.Email)
			return err
		},
	}
}

func newLogoutCommand() *cli.Command {
	return &cli.Command{
		Name:  "logout",
		Usage: "Logout and clear local session",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			if err := st.Service.Logout(ctx); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.Writer, "logged out")
			return err
		},
	}
}
