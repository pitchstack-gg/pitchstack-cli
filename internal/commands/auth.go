package commands

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/pitchstack-gg/pitchstack-cli/internal/pitchstack"

	"github.com/urfave/cli/v3"
)

func newLoginCommand() *cli.Command {
	return &cli.Command{
		Name:  "login",
		Usage: "Login with email and password",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "email", Usage: "Account email"},
			&cli.StringFlag{Name: "password", Usage: "Account password (discouraged; will appear in shell history)"},
			&cli.StringFlag{Name: "device", Usage: "Optional device info label"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			email := strings.TrimSpace(cmd.String("email"))
			password := strings.TrimSpace(cmd.String("password"))
			device := strings.TrimSpace(cmd.String("device"))

			reader := bufio.NewReader(cmd.Reader)
			if email == "" {
				fmt.Fprint(cmd.ErrWriter, "Email: ")
				line, _ := reader.ReadString('\n')
				email = strings.TrimSpace(line)
			}
			if password == "" {
				fmt.Fprint(cmd.ErrWriter, "Password (will echo): ")
				line, _ := reader.ReadString('\n')
				password = strings.TrimSpace(line)
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

			_, err = fmt.Fprintf(cmd.Writer, "logged in as %s (%s)\n", sess.Username, sess.UserID)
			return err
		},
	}
}

func newSignupCommand() *cli.Command {
	return &cli.Command{
		Name:  "signup",
		Usage: "Create a new account",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "email", Usage: "Account email"},
			&cli.StringFlag{Name: "username", Usage: "Optional username"},
			&cli.StringFlag{Name: "password", Usage: "Account password (discouraged; will appear in shell history)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			email := strings.TrimSpace(cmd.String("email"))
			username := strings.TrimSpace(cmd.String("username"))
			password := strings.TrimSpace(cmd.String("password"))

			reader := bufio.NewReader(cmd.Reader)
			if email == "" {
				fmt.Fprint(cmd.ErrWriter, "Email: ")
				line, _ := reader.ReadString('\n')
				email = strings.TrimSpace(line)
			}
			if username == "" {
				fmt.Fprint(cmd.ErrWriter, "Username (optional): ")
				line, _ := reader.ReadString('\n')
				username = strings.TrimSpace(line)
			}
			if password == "" {
				fmt.Fprint(cmd.ErrWriter, "Password (will echo): ")
				line, _ := reader.ReadString('\n')
				password = strings.TrimSpace(line)
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

			_, err = fmt.Fprintf(cmd.Writer, "account created; logged in as %s (%s)\n", sess.Username, sess.UserID)
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

			me, err := st.Service.Me(ctx)
			if err != nil {
				return err
			}
			if me == nil || me.User == nil {
				_, err := fmt.Fprintln(cmd.Writer, "not logged in")
				return err
			}
			_, err = fmt.Fprintf(cmd.Writer, "%s (%s) <%s>\n", me.User.Username, me.User.UserID, me.User.Email)
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
