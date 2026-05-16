package powersync

import (
	"context"
	"errors"
	"strings"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
)

type APIConnector struct {
	Client           *clientv1.Client
	TokenProvider    func(context.Context) (string, error)
	EndpointOverride string
}

func (c *APIConnector) FetchCredentials(ctx context.Context) (*Credentials, error) {
	if c == nil || c.Client == nil {
		return nil, errors.New("api connector client is not configured")
	}
	cfg, err := c.Client.GetPowerSyncClientConfig(ctx)
	if err != nil {
		return nil, err
	}
	token := ""
	if c.TokenProvider != nil {
		token, err = c.TokenProvider(ctx)
		if err != nil {
			return nil, err
		}
	}
	endpoint := strings.TrimSpace(c.EndpointOverride)
	if endpoint == "" && cfg != nil {
		endpoint = strings.TrimSpace(cfg.PowerSyncURL)
	}
	if endpoint == "" {
		return nil, errors.New("powersync endpoint is empty")
	}
	out := &Credentials{Endpoint: endpoint, Token: strings.TrimSpace(token)}
	if cfg != nil {
		out.SyncEpoch = strings.TrimSpace(cfg.SyncEpoch)
	}
	return out, nil
}

func (c *APIConnector) UploadCrud(ctx context.Context, deviceID string, entries []clientv1.CrudEntry) (*clientv1.UploadCrudResponse, error) {
	if c == nil || c.Client == nil {
		return nil, errors.New("api connector client is not configured")
	}
	return c.Client.UploadCrud(ctx, &clientv1.UploadCrudRequest{
		DeviceID: strings.TrimSpace(deviceID),
		Entries:  entries,
	})
}
