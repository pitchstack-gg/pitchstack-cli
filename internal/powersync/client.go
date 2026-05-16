package powersync

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	Store      *Store
	Connector  Connector
	HTTPClient *http.Client
}

func (c *Client) Initialize(ctx context.Context) (*InitResult, error) {
	if c == nil || c.Store == nil {
		return nil, errors.New("powersync store is not configured")
	}
	if c.Connector == nil {
		return nil, errors.New("powersync connector is not configured")
	}
	creds, err := c.Connector.FetchCredentials(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := c.Store.EnsureSyncEpoch(ctx, creds.SyncEpoch); err != nil {
		return nil, err
	}
	deviceID, err := c.Store.EnsureDeviceID(ctx)
	if err != nil {
		return nil, err
	}
	return &InitResult{Path: c.Store.Path(), DeviceID: deviceID, Endpoint: creds.Endpoint, SyncEpoch: creds.SyncEpoch}, nil
}

func (c *Client) PullOnce(ctx context.Context) error {
	if c == nil || c.Store == nil || c.Connector == nil {
		return errors.New("powersync client is not configured")
	}
	creds, err := c.Connector.FetchCredentials(ctx)
	if err != nil {
		return err
	}
	if _, err := c.Store.EnsureSyncEpoch(ctx, creds.SyncEpoch); err != nil {
		return err
	}
	positions, err := c.Store.BucketPositions(ctx)
	if err != nil {
		return err
	}
	resp, err := c.openStream(ctx, creds, positions)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.consumeStream(ctx, resp.Body)
}

func (c *Client) Watch(ctx context.Context) error {
	backoff := time.Second
	for {
		if err := c.UploadPending(ctx); err != nil && ctx.Err() == nil {
			// Keep watching; pending writes remain in the local outbox.
		}
		err := c.PullOnce(ctx)
		if ctx.Err() != nil {
			return nil
		}
		if err == nil {
			backoff = time.Second
			continue
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (c *Client) UploadPending(ctx context.Context) error {
	if c == nil || c.Store == nil || c.Connector == nil {
		return errors.New("powersync client is not configured")
	}
	deviceID, err := c.Store.EnsureDeviceID(ctx)
	if err != nil {
		return err
	}
	for {
		entries, err := c.Store.NextCrudBatch(ctx, 100)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			return nil
		}
		resp, err := c.Connector.UploadCrud(ctx, deviceID, entries)
		if err != nil {
			return err
		}
		if resp == nil {
			return errors.New("empty upload CRUD response")
		}
		if err := c.Store.MarkCrudUploaded(ctx, resp.Results, resp.WriteCheckpoint); err != nil {
			return err
		}
	}
}

func (c *Client) openStream(ctx context.Context, creds *Credentials, buckets []BucketPosition) (*http.Response, error) {
	endpoint := streamEndpoint(strings.TrimSpace(creds.Endpoint))
	if endpoint == "" {
		return nil, errors.New("powersync endpoint is empty")
	}
	payloadBuckets := make([]map[string]string, 0, len(buckets))
	for _, bucket := range buckets {
		payloadBuckets = append(payloadBuckets, map[string]string{
			"name":  bucket.Bucket,
			"after": bucket.OpID,
		})
	}
	deviceID := ""
	if c.Store != nil {
		deviceID, _ = c.Store.EnsureDeviceID(ctx)
	}
	payload := map[string]any{
		"buckets":          payloadBuckets,
		"include_checksum": true,
		"raw_data":         true,
		"client_id":        deviceID,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/x-ndjson")
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(creds.Token) != "" {
		req.Header.Set("Authorization", "Token "+strings.TrimSpace(creds.Token))
	}
	req.Header.Set("X-User-Agent", "pitchstack-cli-powersync-go")
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 0}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("powersync stream failed: %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return resp, nil
}

func streamEndpoint(endpoint string) string {
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return strings.TrimSpace(endpoint)
	}
	u.Path = strings.TrimRight(u.Path, "/")
	if u.Path == "" {
		u.Path = "/sync/stream"
	} else if !strings.HasSuffix(u.Path, "/sync/stream") {
		u.Path += "/sync/stream"
	}
	return u.String()
}

func (c *Client) consumeStream(ctx context.Context, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	var pending []Operation
	var checkpoint string
	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil
		}
		line := normalizeStreamLine(scanner.Text())
		if line == "" {
			continue
		}
		msg, err := parseMessage(line)
		if err != nil {
			return err
		}
		if msg.Checkpoint != "" {
			checkpoint = msg.Checkpoint
		}
		pending = append(pending, msg.Operations...)
		if msg.Complete {
			if err := c.Store.ApplyOperations(ctx, checkpoint, pending); err != nil {
				return err
			}
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(pending) > 0 || checkpoint != "" {
		return c.Store.ApplyOperations(ctx, checkpoint, pending)
	}
	return nil
}

func normalizeStreamLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, ":") {
		return ""
	}
	if strings.HasPrefix(line, "data:") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	}
	return line
}

type parsedMessage struct {
	Checkpoint string
	Complete   bool
	Operations []Operation
}

func parseMessage(line string) (*parsedMessage, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return nil, fmt.Errorf("parse powersync stream line: %w", err)
	}
	if checkpoint, ok := raw["checkpoint"].(map[string]any); ok {
		return &parsedMessage{Checkpoint: firstString(checkpoint, "last_op_id", "checkpoint", "checkpoint_id", "checkpointId")}, nil
	}
	if complete, ok := raw["checkpoint_complete"].(map[string]any); ok {
		return &parsedMessage{
			Checkpoint: firstString(complete, "last_op_id", "checkpoint", "checkpoint_id", "checkpointId"),
			Complete:   true,
		}, nil
	}
	if _, ok := raw["checkpoint_complete"]; ok {
		return &parsedMessage{Complete: true}, nil
	}
	if data, ok := raw["data"].(map[string]any); ok {
		out := &parsedMessage{}
		bucket := firstString(data, "bucket", "name")
		if rows, ok := data["data"].([]any); ok {
			for _, row := range rows {
				if m, ok := row.(map[string]any); ok {
					out.Operations = append(out.Operations, operationFromMap(m, bucket, firstString(m, "checksum")))
				}
			}
		}
		return out, nil
	}
	kind := strings.ToLower(firstString(raw, "type", "event", "kind"))
	out := &parsedMessage{}
	out.Checkpoint = firstString(raw, "checkpoint", "checkpoint_id", "checkpointId", "write_checkpoint", "writeCheckpoint")
	if strings.Contains(kind, "complete") {
		out.Complete = true
	}
	bucket := firstString(raw, "bucket", "bucket_id", "bucketId")
	checksum := firstString(raw, "checksum")
	if rows, ok := raw["rows"].([]any); ok {
		for _, row := range rows {
			if m, ok := row.(map[string]any); ok {
				out.Operations = append(out.Operations, operationFromMap(m, bucket, checksum))
			}
		}
	}
	if dataRows, ok := raw["data"].([]any); ok {
		for _, row := range dataRows {
			if m, ok := row.(map[string]any); ok {
				out.Operations = append(out.Operations, operationFromMap(m, bucket, checksum))
			}
		}
	} else if _, hasData := raw["data"]; hasData || firstString(raw, "table", "type", "type_name", "table_name") != "" || firstString(raw, "id") != "" {
		op := operationFromMap(raw, bucket, checksum)
		if op.Table != "" && op.ID != "" {
			out.Operations = append(out.Operations, op)
		}
	}
	if kind == "checkpoint_complete" || kind == "checkpoint-complete" {
		out.Complete = true
	}
	return out, nil
}

func operationFromMap(raw map[string]any, fallbackBucket, fallbackChecksum string) Operation {
	op := Operation{
		Bucket:   coalesce(firstString(raw, "bucket", "bucket_id", "bucketId"), fallbackBucket),
		OpID:     valueString(firstValue(raw, "op_id", "opId", "client_id", "clientId")),
		Op:       firstString(raw, "op", "operation", "update_type", "updateType"),
		Table:    firstString(raw, "table", "table_name", "tableName", "type", "type_name", "typeName", "object_type", "objectType"),
		ID:       firstString(raw, "id", "row_id", "rowId", "object_id", "objectId"),
		Checksum: coalesce(firstString(raw, "checksum"), fallbackChecksum),
	}
	if strings.EqualFold(op.Op, "data") || strings.EqualFold(op.Op, "row") {
		op.Op = firstString(raw, "operation", "update_type", "updateType")
	}
	if op.Op == "" && strings.EqualFold(firstString(raw, "remove"), "true") {
		op.Op = "REMOVE"
	}
	if data, ok := raw["data"].(map[string]any); ok {
		op.Data = data
	} else if data, ok := raw["opData"].(map[string]any); ok {
		op.Data = data
	} else if data, ok := raw["op_data"].(map[string]any); ok {
		op.Data = data
	} else if data, ok := raw["values"].(map[string]any); ok {
		op.Data = data
	} else if data, ok := decodeEmbeddedData(firstValue(raw, "data", "opData", "op_data", "values")); ok {
		op.Data = data
	} else {
		op.Data = map[string]any{}
		for key, value := range raw {
			if isProtocolField(key) {
				continue
			}
			op.Data[key] = value
		}
	}
	if op.ID == "" {
		op.ID = firstString(op.Data, "id")
	}
	return op
}

func decodeEmbeddedData(value any) (map[string]any, bool) {
	raw := strings.TrimSpace(valueString(value))
	if raw == "" || !strings.HasPrefix(raw, "{") {
		return nil, false
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, false
	}
	return out, true
}

func firstString(m map[string]any, keys ...string) string {
	return valueString(firstValue(m, keys...))
}

func firstValue(m map[string]any, keys ...string) any {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return nil
}

func valueString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func coalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func isProtocolField(key string) bool {
	switch key {
	case "bucket", "bucket_id", "bucketId", "op_id", "opId", "client_id", "clientId", "op", "operation", "type", "table", "table_name", "tableName", "type_name", "typeName", "row_id", "rowId", "object_id", "objectId", "object_type", "objectType", "subkey", "checksum", "data", "values", "opData", "op_data":
		return true
	default:
		return false
	}
}
