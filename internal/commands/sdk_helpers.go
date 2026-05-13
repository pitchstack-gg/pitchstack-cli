package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
	"github.com/urfave/cli/v3"
)

type sdkCall[TReq any] func(context.Context, *clientv1.Client, *TReq) (any, error)

func requestFileFlag() cli.Flag {
	return &cli.StringFlag{Name: "file", Usage: "JSON request file, or - for stdin"}
}

func pageFlags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{Name: "page-size", Usage: "Page size"},
		&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
	}
}

func repeatedIDsFlag(name, usage string) cli.Flag {
	return &cli.StringSliceFlag{Name: name, Usage: usage}
}

func newSDKCommand[TReq any](name string, usage string, flags []cli.Flag, authenticated bool, apply func(*cli.Command, *TReq) error, call sdkCall[TReq]) *cli.Command {
	allFlags := append([]cli.Flag{requestFileFlag()}, flags...)
	return &cli.Command{
		Name:  name,
		Usage: usage,
		Flags: allFlags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			var req TReq
			if err := readRequestFile(cmd, &req); err != nil {
				return err
			}
			if apply != nil {
				if err := apply(cmd, &req); err != nil {
					return err
				}
			}
			return withSDKClient(ctx, cmd, authenticated, func(c *clientv1.Client) (any, error) {
				return call(ctx, c, &req)
			})
		},
	}
}

func newSDKNoRequestCommand(name string, usage string, authenticated bool, call func(context.Context, *clientv1.Client) (any, error)) *cli.Command {
	return &cli.Command{
		Name:  name,
		Usage: usage,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withSDKClient(ctx, cmd, authenticated, func(c *clientv1.Client) (any, error) {
				return call(ctx, c)
			})
		},
	}
}

func withSDKClient(ctx context.Context, cmd *cli.Command, authenticated bool, fn func(*clientv1.Client) (any, error)) error {
	st, err := getState(ctx)
	if err != nil {
		return err
	}
	var c *clientv1.Client
	if authenticated {
		c, err = st.Service.AuthenticatedClient()
	} else {
		c, err = st.Service.UnauthenticatedClient()
	}
	if err != nil {
		return err
	}
	resp, err := fn(c)
	if err != nil {
		return err
	}
	return writeJSON(cmd.Writer, resp)
}

func readRequestFile[T any](cmd *cli.Command, out *T) error {
	path := strings.TrimSpace(cmd.String("file"))
	if path == "" {
		return nil
	}
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(cmd.Reader)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return fmt.Errorf("read --file: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode --file JSON: %w", err)
	}
	return nil
}

func setStringFlag(cmd *cli.Command, name string, dst *string) {
	if cmd.IsSet(name) {
		*dst = strings.TrimSpace(cmd.String(name))
	}
}

func setStringSliceFlag(cmd *cli.Command, name string, dst *[]string) {
	if cmd.IsSet(name) {
		*dst = trimStrings(cmd.StringSlice(name))
	}
}

func setInt32Flag(cmd *cli.Command, name string, dst **int32) {
	if cmd.IsSet(name) {
		v := int32(cmd.Int(name))
		*dst = &v
	}
}

func setInt64Flag(cmd *cli.Command, name string, dst **int64) {
	if cmd.IsSet(name) {
		v := int64(cmd.Int(name))
		*dst = &v
	}
}

func setFloat64Flag(cmd *cli.Command, name string, dst **float64) {
	if cmd.IsSet(name) {
		v := cmd.Float64(name)
		*dst = &v
	}
}

func setBoolFlag(cmd *cli.Command, name string, dst **bool) {
	if cmd.IsSet(name) {
		v := cmd.Bool(name)
		*dst = &v
	}
}

func setStringPtrFlag(cmd *cli.Command, name string, dst **string) {
	if cmd.IsSet(name) {
		v := strings.TrimSpace(cmd.String(name))
		*dst = &v
	}
}

func setTimeFlag(cmd *cli.Command, name string, dst **time.Time) error {
	if !cmd.IsSet(name) {
		return nil
	}
	v, err := time.Parse(time.RFC3339, strings.TrimSpace(cmd.String(name)))
	if err != nil {
		return fmt.Errorf("--%s must be RFC3339: %w", name, err)
	}
	*dst = &v
	return nil
}

func setPageFlags(cmd *cli.Command, pageSize **int32, nextToken *string) {
	setInt32Flag(cmd, "page-size", pageSize)
	setStringFlag(cmd, "next-token", nextToken)
}

func trimStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func splitCSV(values []string) []string {
	var out []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

func parseInt64Flag(cmd *cli.Command, name string) (*int64, error) {
	if !cmd.IsSet(name) {
		return nil, nil
	}
	raw := strings.TrimSpace(cmd.String(name))
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("--%s must be an integer: %w", name, err)
	}
	return &v, nil
}

func parseFloatFlag(cmd *cli.Command, name string) (*float64, error) {
	if !cmd.IsSet(name) {
		return nil, nil
	}
	raw := strings.TrimSpace(cmd.String(name))
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, fmt.Errorf("--%s must be a number: %w", name, err)
	}
	return &v, nil
}

func uploadFileToSignedURL(ctx context.Context, httpClient *http.Client, uploadURL string, requiredHeaders map[string]string, filePath string, fallbackContentType string) error {
	uploadURL = strings.TrimSpace(uploadURL)
	if uploadURL == "" {
		return fmt.Errorf("missing upload URL")
	}
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open upload file: %w", err)
	}
	defer f.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, f)
	if err != nil {
		return err
	}
	for k, v := range requiredHeaders {
		if strings.TrimSpace(k) != "" {
			req.Header.Set(k, v)
		}
	}
	if fallbackContentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", fallbackContentType)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("upload failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}
