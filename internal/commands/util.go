package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

func boolPtr(cmd *cli.Command, name string) *bool {
	if !cmd.IsSet(name) {
		return nil
	}
	v := cmd.Bool(name)
	return &v
}

func int32Ptr(cmd *cli.Command, name string) *int32 {
	if !cmd.IsSet(name) {
		return nil
	}
	v := int32(cmd.Int(name))
	return &v
}

func float64Ptr(cmd *cli.Command, name string) *float64 {
	if !cmd.IsSet(name) {
		return nil
	}
	v := cmd.Float64(name)
	return &v
}

func parseInt32(s string) (*int32, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil, err
	}
	vs := int32(v)
	return &vs, nil
}

func yesFlag() cli.Flag {
	return &cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "Skip confirmation prompt"}
}

func confirmAction(cmd *cli.Command, verb, resource, id string) error {
	if cmd.Bool("yes") {
		return nil
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return cli.Exit("missing id", 2)
	}
	if _, err := fmt.Fprintf(cmd.ErrWriter, "%s %s %s? Type yes to continue: ", verb, resource, id); err != nil {
		return err
	}
	line, err := readLine(cmd.Reader)
	if err != nil {
		return err
	}
	if strings.TrimSpace(line) != "yes" {
		return cli.Exit("aborted", 1)
	}
	return nil
}

func readPrompt(cmd *cli.Command, prompt string) (string, error) {
	if _, err := fmt.Fprint(cmd.ErrWriter, prompt); err != nil {
		return "", err
	}
	line, err := readLine(cmd.Reader)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func readSecret(cmd *cli.Command, prompt string) (string, error) {
	if _, err := fmt.Fprint(cmd.ErrWriter, prompt); err != nil {
		return "", err
	}
	if f, ok := cmd.Reader.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		data, err := term.ReadPassword(int(f.Fd()))
		_, _ = fmt.Fprintln(cmd.ErrWriter)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	line, err := readLine(cmd.Reader)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func readLine(r io.Reader) (string, error) {
	var b strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if buf[0] == '\n' {
				return b.String(), nil
			}
			b.WriteByte(buf[0])
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return b.String(), nil
			}
			return "", err
		}
	}
}
