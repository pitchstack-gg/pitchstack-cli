package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
)

func newAuthCommand() *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Authentication helpers",
		Commands: []*cli.Command{
			newAuthStatusCommand(),
			newAuthMeCommand(),
			newAuthAPIKeysCommand(),
			newAuthPasswordCommand(),
			newAuthMethodsCommand(),
			newAuthOAuthCommand(),
			newAuthPatreonCommand(),
			newAuthInternalCommand(),
			newAuthTokensCommand(),
			newAuthUsersCommand(),
			newAuthPasskeysCommand(),
		},
	}
}

func newAuthStatusCommand() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Show auth status and token expiry",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.Writer, "api: %s\n", strings.TrimSpace(st.Profile.BaseURL))
			if strings.TrimSpace(st.Profile.OAuthBaseURL) != "" {
				_, _ = fmt.Fprintf(cmd.Writer, "auth: %s\n", strings.TrimSpace(st.Profile.OAuthBaseURL))
			}
			if st.Sessions != nil {
				_, _ = fmt.Fprintf(cmd.Writer, "session: %s\n", st.Sessions.Path())
			}

			sess, err := st.Sessions.Load()
			if err != nil {
				return err
			}
			if sess == nil {
				_, _ = fmt.Fprintln(cmd.Writer, "status: not logged in")
				return nil
			}

			userLabel := strings.TrimSpace(sess.UserID)
			if strings.TrimSpace(sess.Username) != "" {
				userLabel = fmt.Sprintf("%s (%s)", strings.TrimSpace(sess.Username), strings.TrimSpace(sess.UserID))
			}

			_, _ = fmt.Fprintf(cmd.Writer, "user: %s\n", userLabel)
			if strings.TrimSpace(sess.RefreshToken) != "" {
				_, _ = fmt.Fprintln(cmd.Writer, "refresh token: present")
			} else {
				_, _ = fmt.Fprintln(cmd.Writer, "refresh token: missing")
			}

			exp := sess.AccessTokenExpiresAt
			if exp.IsZero() {
				_, _ = fmt.Fprintln(cmd.Writer, "access token: missing/unknown expiry")
				return nil
			}

			until := time.Until(exp)
			if until <= 0 {
				_, _ = fmt.Fprintf(cmd.Writer, "access token: expired (%s)\n", exp.Format(time.RFC3339))
				return nil
			}
			_, _ = fmt.Fprintf(cmd.Writer, "access token: valid for %s (until %s)\n", until.Truncate(time.Second), exp.Format(time.RFC3339))
			return nil
		},
	}
}
