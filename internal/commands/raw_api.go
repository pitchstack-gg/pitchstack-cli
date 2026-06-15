package commands

import (
	"context"
	"strings"

	"github.com/urfave/cli/v3"
)

func callAuthenticatedJSON(ctx context.Context, cmd *cli.Command, method string, path string, payload any, out any) error {
	st, err := getState(ctx)
	if err != nil {
		return err
	}
	return st.Service.DoJSON(ctx, method, path, payload, out, true)
}

func writeAuthenticatedJSON(ctx context.Context, cmd *cli.Command, method string, path string, payload any) error {
	var resp map[string]any
	if err := callAuthenticatedJSON(ctx, cmd, method, path, payload, &resp); err != nil {
		return err
	}
	return writeJSON(cmd.Writer, resp)
}

func readObjectPayload(cmd *cli.Command) (map[string]any, error) {
	payload := map[string]any{}
	if err := readRequestFile(cmd, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func setPayloadStringFlag(cmd *cli.Command, flag string, key string, payload map[string]any) {
	if cmd.IsSet(flag) {
		payload[key] = strings.TrimSpace(cmd.String(flag))
	}
}

func setPayloadBoolFlag(cmd *cli.Command, flag string, key string, payload map[string]any) {
	if cmd.IsSet(flag) {
		payload[key] = cmd.Bool(flag)
	}
}

func setPayloadIntFlag(cmd *cli.Command, flag string, key string, payload map[string]any) {
	if cmd.IsSet(flag) {
		payload[key] = cmd.Int(flag)
	}
}
