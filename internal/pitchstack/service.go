package pitchstack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"

	"github.com/pitchstack-gg/pitchstack-cli/internal/session"
)

type ServiceDeps struct {
	BaseURL        string
	TimeoutSeconds int
	Sessions       *session.Store
}

type Service struct {
	baseURL string
	timeout time.Duration
	store   *session.Store

	refreshMu sync.Mutex
}

func NewService(deps ServiceDeps) *Service {
	timeout := 30 * time.Second
	if deps.TimeoutSeconds > 0 {
		timeout = time.Duration(deps.TimeoutSeconds) * time.Second
	}
	return &Service{
		baseURL: strings.TrimSpace(deps.BaseURL),
		timeout: timeout,
		store:   deps.Sessions,
	}
}

func (s *Service) client(withCreds bool) (*clientv1.Client, error) {
	opts := []clientv1.ClientOpt{
		clientv1.WithBaseURL(s.baseURL),
		clientv1.WithHTTPTimeout(s.timeout),
	}
	if withCreds {
		opts = append(opts, clientv1.WithCredentialProvider(clientv1.CredentialProviderFunc(s.credential)))
	}
	return clientv1.NewClient(opts...)
}

func (s *Service) Login(ctx context.Context, in LoginInput) (*session.Session, error) {
	if s.store == nil {
		return nil, errors.New("session store is not configured")
	}
	c, err := s.client(false)
	if err != nil {
		return nil, err
	}
	resp, err := c.Login(ctx, &clientv1.LoginRequest{
		Email:      in.Email,
		Password:   in.Password,
		DeviceInfo: in.DeviceInfo,
	})
	if err != nil {
		return nil, err
	}

	sess := &session.Session{
		BaseURL:              s.baseURL,
		UserID:               resp.UserID,
		Username:             resp.Username,
		Roles:                resp.Roles,
		AccessToken:          resp.AccessToken,
		RefreshToken:         resp.RefreshToken,
		AccessTokenExpiresAt: derefTime(resp.AccessTokenExpiresAt),
	}
	if err := s.store.Save(sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *Service) Logout(ctx context.Context) error {
	if s.store == nil {
		return errors.New("session store is not configured")
	}
	sess, err := s.store.Load()
	if err != nil {
		return err
	}
	if sess == nil || strings.TrimSpace(sess.RefreshToken) == "" {
		return s.store.Clear()
	}

	c, err := s.client(false)
	if err != nil {
		return err
	}
	_, _ = c.Logout(ctx, &clientv1.LogoutRequest{RefreshToken: sess.RefreshToken})

	return s.store.Clear()
}

func (s *Service) Me(ctx context.Context) (*clientv1.MeResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.Me(ctx)
}

func (s *Service) ListCollections(ctx context.Context, request *clientv1.ListCollectionsRequest) (*clientv1.ListCollectionsResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.ListCollections(ctx, request)
}

func (s *Service) CreateCollection(ctx context.Context, request *clientv1.CreateCollectionRequest) (*clientv1.CreateCollectionResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.CreateCollection(ctx, request)
}

func (s *Service) GetCollection(ctx context.Context, request *clientv1.GetCollectionRequest) (*clientv1.GetCollectionResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.GetCollection(ctx, request)
}

func (s *Service) UpdateCollection(ctx context.Context, request *clientv1.UpdateCollectionRequest) (*clientv1.UpdateCollectionResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.UpdateCollection(ctx, request)
}

func (s *Service) UpdateCollectionVisibility(ctx context.Context, request *clientv1.UpdateCollectionVisibilityRequest) (*clientv1.UpdateCollectionVisibilityResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.UpdateCollectionVisibility(ctx, request)
}

func (s *Service) DeleteCollection(ctx context.Context, request *clientv1.DeleteCollectionRequest) (*clientv1.DeleteCollectionResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.DeleteCollection(ctx, request)
}

func (s *Service) ListCollectionItems(ctx context.Context, request *clientv1.ListCollectionItemsRequest) (*clientv1.ListCollectionItemsResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.ListCollectionItems(ctx, request)
}

func (s *Service) GetCollectionItem(ctx context.Context, request *clientv1.GetCollectionItemRequest) (*clientv1.GetCollectionItemResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.GetCollectionItem(ctx, request)
}

func (s *Service) CreateCollectionItem(ctx context.Context, request *clientv1.CreateCollectionItemRequest) (*clientv1.CreateCollectionItemResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.CreateCollectionItem(ctx, request)
}

func (s *Service) UpdateCollectionItem(ctx context.Context, request *clientv1.UpdateCollectionItemRequest) (*clientv1.UpdateCollectionItemResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.UpdateCollectionItem(ctx, request)
}

func (s *Service) DeleteCollectionItem(ctx context.Context, request *clientv1.DeleteCollectionItemRequest) (*clientv1.DeleteCollectionItemResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.DeleteCollectionItem(ctx, request)
}

func (s *Service) GetChangeSet(ctx context.Context, request *clientv1.GetChangeSetRequest) (*clientv1.GetChangeSetResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.GetChangeSet(ctx, request)
}

func (s *Service) BatchApplyChanges(ctx context.Context, request *clientv1.BatchApplyChangesRequest) (*clientv1.BatchApplyChangesResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.BatchApplyChanges(ctx, request)
}

func (s *Service) UpdateSubscriptions(ctx context.Context, request *clientv1.UpdateSubscriptionsRequest) (*clientv1.UpdateSubscriptionsResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.UpdateSubscriptions(ctx, request)
}

func (s *Service) ListSubscriptions(ctx context.Context) (*clientv1.ListSubscriptionsResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.ListSubscriptions(ctx)
}

func (s *Service) SearchCards(ctx context.Context, request *clientv1.SearchCardsRequest) (*clientv1.SearchCardsResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.SearchCards(ctx, request)
}

func (s *Service) GetCard(ctx context.Context, request *clientv1.GetCardRequest) (*clientv1.GetCardResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.GetCard(ctx, request)
}

func (s *Service) ListPrintings(ctx context.Context, request *clientv1.ListPrintingsRequest) (*clientv1.ListPrintingsResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.ListPrintings(ctx, request)
}

func (s *Service) GetPrinting(ctx context.Context, request *clientv1.GetPrintingRequest) (*clientv1.GetPrintingResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.GetPrinting(ctx, request)
}

func (s *Service) ListPrintingsForSetNumber(ctx context.Context, request *clientv1.ListPrintingsForSetNumberRequest) (*clientv1.ListPrintingsForSetNumberResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.ListPrintingsForSetNumber(ctx, request)
}

func (s *Service) GetProduct(ctx context.Context, request *clientv1.GetProductRequest) (*clientv1.GetProductResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.GetProduct(ctx, request)
}

func (s *Service) GetDataSnapshot(ctx context.Context, request *clientv1.GetDataSnapshotRequest) (*clientv1.GetDataSnapshotResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.GetDataSnapshot(ctx, request)
}

func (s *Service) GetProfile(ctx context.Context, request *clientv1.GetProfileRequest) (*clientv1.GetProfileResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.GetProfile(ctx, request)
}

func (s *Service) UpdateProfile(ctx context.Context, request *clientv1.UpdateProfileRequest) (*clientv1.UpdateProfileResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.UpdateProfile(ctx, request)
}

func (s *Service) GetProfileSettings(ctx context.Context, request *clientv1.GetProfileSettingsRequest) (*clientv1.GetProfileSettingsResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.GetProfileSettings(ctx, request)
}

func (s *Service) UpdateProfileSettings(ctx context.Context, request *clientv1.UpdateProfileSettingsRequest) (*clientv1.UpdateProfileSettingsResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.UpdateProfileSettings(ctx, request)
}

func (s *Service) SetAvatarURL(ctx context.Context, request *clientv1.SetAvatarURLRequest) (*clientv1.SetAvatarURLResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.SetAvatarURL(ctx, request)
}

func (s *Service) GetSocialProfiles(ctx context.Context, request *clientv1.GetSocialProfilesRequest) (*clientv1.GetSocialProfilesResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.GetSocialProfiles(ctx, request)
}

func (s *Service) UpsertSocialProfile(ctx context.Context, request *clientv1.UpsertSocialProfileRequest) (*clientv1.UpsertSocialProfileResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.UpsertSocialProfile(ctx, request)
}

func (s *Service) RemoveSocialProfile(ctx context.Context, request *clientv1.RemoveSocialProfileRequest) (*clientv1.RemoveSocialProfileResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.RemoveSocialProfile(ctx, request)
}

type LoginInput struct {
	Email      string
	Password   string
	DeviceInfo string
}

type SignupInput struct {
	Email    string
	Username string
	Password string
}

func (s *Service) Signup(ctx context.Context, in SignupInput) (*session.Session, error) {
	if s.store == nil {
		return nil, errors.New("session store is not configured")
	}
	payload := map[string]any{
		"email":    in.Email,
		"password": in.Password,
	}
	if strings.TrimSpace(in.Username) != "" {
		payload["username"] = strings.TrimSpace(in.Username)
	}

	var resp struct {
		UserID               string     `json:"userId,omitempty"`
		Username             string     `json:"username,omitempty"`
		AccessToken          string     `json:"accessToken,omitempty"`
		RefreshToken         string     `json:"refreshToken,omitempty"`
		AccessTokenExpiresAt *time.Time `json:"accessTokenExpiresAt,omitempty"`
		Roles                []string   `json:"roles,omitempty"`
	}
	if err := s.postJSON(ctx, "/v1/auth/register", payload, &resp, nil); err != nil {
		return nil, err
	}

	sess := &session.Session{
		BaseURL:              s.baseURL,
		UserID:               resp.UserID,
		Username:             resp.Username,
		Roles:                resp.Roles,
		AccessToken:          resp.AccessToken,
		RefreshToken:         resp.RefreshToken,
		AccessTokenExpiresAt: derefTime(resp.AccessTokenExpiresAt),
	}
	if err := s.store.Save(sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *Service) credential(ctx context.Context) (*clientv1.Credential, error) {
	if s.store == nil {
		return nil, nil
	}
	sess, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, nil
	}

	token := strings.TrimSpace(sess.AccessToken)
	if token == "" || tokenExpiringSoon(sess.AccessTokenExpiresAt) {
		if err := s.refresh(ctx); err != nil {
			return nil, err
		}
		sess, err = s.store.Load()
		if err != nil {
			return nil, err
		}
		if sess == nil {
			return nil, nil
		}
		token = strings.TrimSpace(sess.AccessToken)
	}

	if token == "" {
		return nil, nil
	}
	return &clientv1.Credential{Header: "Authorization", Value: "Bearer " + token}, nil
}

func (s *Service) refresh(ctx context.Context) error {
	if s.store == nil {
		return errors.New("session store is not configured")
	}
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	latest, err := s.store.Load()
	if err != nil {
		return err
	}
	if latest == nil {
		return nil
	}
	if strings.TrimSpace(latest.AccessToken) != "" && !tokenExpiringSoon(latest.AccessTokenExpiresAt) {
		return nil
	}
	if strings.TrimSpace(latest.RefreshToken) == "" {
		return errors.New("not logged in (missing refresh token)")
	}

	c, err := s.client(false)
	if err != nil {
		return err
	}
	resp, err := c.RefreshToken(ctx, &clientv1.RefreshTokenRequest{RefreshToken: latest.RefreshToken})
	if err != nil {
		return err
	}

	latest.AccessToken = resp.AccessToken
	if strings.TrimSpace(resp.RefreshToken) != "" {
		latest.RefreshToken = resp.RefreshToken
	}
	latest.AccessTokenExpiresAt = derefTime(resp.AccessTokenExpiresAt)
	return s.store.Save(latest)
}

func tokenExpiringSoon(exp time.Time) bool {
	if exp.IsZero() {
		return true
	}
	return time.Until(exp) <= 60*time.Second
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return t.UTC()
}

func (s *Service) postJSON(ctx context.Context, path string, payload any, out any, headers http.Header) error {
	base, err := url.Parse(s.baseURL)
	if err != nil {
		return fmt.Errorf("parse base url: %w", err)
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	rel, err := url.Parse(path)
	if err != nil {
		return fmt.Errorf("parse path: %w", err)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base.ResolveReference(rel).String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, vals := range headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}

	httpClient := &http.Client{Timeout: s.timeout}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("api error (%d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
