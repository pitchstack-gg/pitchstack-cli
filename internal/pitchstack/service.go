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
	httpClient := &http.Client{
		Timeout: s.timeout,
		Transport: &authRetryRoundTripper{
			base: http.DefaultTransport,
			svc:  s,
		},
	}
	opts := []clientv1.ClientOpt{
		clientv1.WithBaseURL(s.baseURL),
		clientv1.WithHTTPClient(httpClient),
	}
	if withCreds {
		opts = append(opts, clientv1.WithCredentialProvider(clientv1.CredentialProviderFunc(s.credential)))
	}
	return clientv1.NewClient(opts...)
}

func (s *Service) AuthenticatedClient() (*clientv1.Client, error) {
	return s.client(true)
}

func (s *Service) UnauthenticatedClient() (*clientv1.Client, error) {
	return s.client(false)
}

func (s *Service) HTTPClient() *http.Client {
	return &http.Client{Timeout: s.timeout}
}

func (s *Service) Credential(ctx context.Context) (*clientv1.Credential, error) {
	return s.credential(ctx)
}

func (s *Service) BearerToken(ctx context.Context) (string, error) {
	cred, err := s.credential(ctx)
	if err != nil {
		return "", err
	}
	if cred == nil {
		return "", nil
	}
	value := strings.TrimSpace(cred.Value)
	if strings.EqualFold(strings.TrimSpace(cred.Header), "Authorization") && strings.HasPrefix(strings.ToLower(value), "bearer ") {
		return strings.TrimSpace(value[len("bearer "):]), nil
	}
	return value, nil
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
	return s.saveSessionFromLogin(ctx, loginResult{
		UserID:               resp.UserID,
		Roles:                resp.Roles,
		AccessToken:          resp.AccessToken,
		RefreshToken:         resp.RefreshToken,
		AccessTokenExpiresAt: resp.AccessTokenExpiresAt,
	})
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

func (s *Service) GrantCollectionAccess(ctx context.Context, request *clientv1.GrantCollectionAccessRequest) (*clientv1.GrantCollectionAccessResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.GrantCollectionAccess(ctx, request)
}

func (s *Service) RevokeCollectionAccess(ctx context.Context, request *clientv1.RevokeCollectionAccessRequest) (*clientv1.RevokeCollectionAccessResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.RevokeCollectionAccess(ctx, request)
}

func (s *Service) GetCollectionAccess(ctx context.Context, request *clientv1.GetCollectionAccessRequest) (*clientv1.GetCollectionAccessResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.GetCollectionAccess(ctx, request)
}

func (s *Service) ListCollectionAccessGrants(ctx context.Context, request *clientv1.ListCollectionAccessGrantsRequest) (*clientv1.ListCollectionAccessGrantsResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.ListCollectionAccessGrants(ctx, request)
}

func (s *Service) GetDeckAccess(ctx context.Context, request *clientv1.GetDeckAccessRequest) (*clientv1.GetDeckAccessResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.GetDeckAccess(ctx, request)
}

func (s *Service) ListDeckAccessGrants(ctx context.Context, request *clientv1.ListDeckAccessGrantsRequest) (*clientv1.ListDeckAccessGrantsResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.ListDeckAccessGrants(ctx, request)
}

func (s *Service) ListDecks(ctx context.Context, request *clientv1.ListDecksRequest) (*clientv1.ListDecksResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.ListDecks(ctx, request)
}

func (s *Service) CreateDeck(ctx context.Context, request *clientv1.CreateDeckRequest) (*clientv1.CreateDeckResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.CreateDeck(ctx, request)
}

func (s *Service) CloneDeck(ctx context.Context, request *clientv1.CloneDeckRequest) (*clientv1.CloneDeckResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.CloneDeck(ctx, request)
}

func (s *Service) GetDeck(ctx context.Context, request *clientv1.GetDeckRequest) (*clientv1.GetDeckResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.GetDeck(ctx, request)
}

func (s *Service) UpdateDeck(ctx context.Context, request *clientv1.UpdateDeckRequest) (*clientv1.UpdateDeckResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.UpdateDeck(ctx, request)
}

func (s *Service) UpdateDeckVisibility(ctx context.Context, request *clientv1.UpdateDeckVisibilityRequest) (*clientv1.UpdateDeckVisibilityResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.UpdateDeckVisibility(ctx, request)
}

func (s *Service) DeleteDeck(ctx context.Context, request *clientv1.DeleteDeckRequest) (*clientv1.DeleteDeckResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.DeleteDeck(ctx, request)
}

func (s *Service) SearchDecks(ctx context.Context, request *clientv1.SearchDecksRequest) (*clientv1.SearchDecksResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.SearchDecks(ctx, request)
}

func (s *Service) GrantDeckAccess(ctx context.Context, request *clientv1.GrantDeckAccessRequest) (*clientv1.GrantDeckAccessResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.GrantDeckAccess(ctx, request)
}

func (s *Service) RevokeDeckAccess(ctx context.Context, request *clientv1.RevokeDeckAccessRequest) (*clientv1.RevokeDeckAccessResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.RevokeDeckAccess(ctx, request)
}

func (s *Service) ListDeckVersions(ctx context.Context, request *clientv1.ListDeckVersionsRequest) (*clientv1.ListDeckVersionsResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.ListDeckVersions(ctx, request)
}

func (s *Service) CreateDeckVersion(ctx context.Context, request *clientv1.CreateDeckVersionRequest) (*clientv1.CreateDeckVersionResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.CreateDeckVersion(ctx, request)
}

func (s *Service) GetDeckVersion(ctx context.Context, request *clientv1.GetDeckVersionRequest) (*clientv1.GetDeckVersionResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.GetDeckVersion(ctx, request)
}

func (s *Service) GetDeckVersionHistory(ctx context.Context, request *clientv1.GetDeckVersionHistoryRequest) (*clientv1.GetDeckVersionHistoryResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.GetDeckVersionHistory(ctx, request)
}

func (s *Service) GetDeckVersionNotes(ctx context.Context, request *clientv1.GetDeckVersionNotesRequest) (*clientv1.GetDeckVersionNotesResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.GetDeckVersionNotes(ctx, request)
}

func (s *Service) UpdateDeckVersionNotes(ctx context.Context, request *clientv1.UpdateDeckVersionNotesRequest) (*clientv1.UpdateDeckVersionNotesResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.UpdateDeckVersionNotes(ctx, request)
}

func (s *Service) ListDeckVersionSideboardGuides(ctx context.Context, request *clientv1.ListDeckVersionSideboardGuidesRequest) (*clientv1.ListDeckVersionSideboardGuidesResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.ListDeckVersionSideboardGuides(ctx, request)
}

func (s *Service) UpsertDeckVersionSideboardGuide(ctx context.Context, request *clientv1.UpsertDeckVersionSideboardGuideRequest) (*clientv1.UpsertDeckVersionSideboardGuideResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.UpsertDeckVersionSideboardGuide(ctx, request)
}

func (s *Service) DeleteDeckVersionSideboardGuide(ctx context.Context, request *clientv1.DeleteDeckVersionSideboardGuideRequest) (*clientv1.DeleteDeckVersionSideboardGuideResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.DeleteDeckVersionSideboardGuide(ctx, request)
}

func (s *Service) ListDeckVersionCards(ctx context.Context, request *clientv1.ListDeckVersionCardsRequest) (*clientv1.ListDeckVersionCardsResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.ListDeckVersionCards(ctx, request)
}

func (s *Service) ModifyDeckVersionCard(ctx context.Context, request *clientv1.ModifyDeckVersionCardRequest) (*clientv1.ModifyDeckVersionCardResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.ModifyDeckVersionCard(ctx, request)
}

func (s *Service) DeleteDeckVersion(ctx context.Context, request *clientv1.DeleteDeckVersionRequest) (*clientv1.DeleteDeckVersionResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.DeleteDeckVersion(ctx, request)
}

func (s *Service) BatchGetDecks(ctx context.Context, request *clientv1.BatchGetDecksRequest) (*clientv1.BatchGetDecksResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.BatchGetDecks(ctx, request)
}

func (s *Service) ExportDeck(ctx context.Context, request *clientv1.ExportDeckRequest) (*clientv1.ExportDeckResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.ExportDeck(ctx, request)
}

func (s *Service) ImportDeck(ctx context.Context, request *clientv1.ImportDeckRequest) (*clientv1.ImportDeckResponse, error) {
	c, err := s.client(true)
	if err != nil {
		return nil, err
	}
	return c.ImportDeck(ctx, request)
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

type CLILoginSession struct {
	SessionID        string
	SessionSecret    string
	VerificationPath string
	VerificationURL  string
	ExpiresAt        time.Time
	PollInterval     time.Duration
}

type CLILoginSessionPoll struct {
	Status string
	Login  *CLILoginSessionLogin
}

type CLILoginSessionLogin struct {
	UserID               string
	Roles                []string
	AccessToken          string
	RefreshToken         string
	AccessTokenExpiresAt *time.Time
}

func (s *Service) CreateCLILoginSession(ctx context.Context, oauthBaseURL string) (*CLILoginSession, error) {
	if strings.TrimSpace(oauthBaseURL) == "" {
		return nil, errors.New("oauth base url must not be empty")
	}

	var resp struct {
		SessionID           string     `json:"sessionId,omitempty"`
		SessionSecret       string     `json:"sessionSecret,omitempty"`
		VerificationPath    string     `json:"verificationPath,omitempty"`
		VerificationURL     string     `json:"verificationUrl,omitempty"`
		ExpiresAt           *time.Time `json:"expiresAt,omitempty"`
		PollIntervalSeconds int        `json:"pollIntervalSeconds,omitempty"`
	}
	if err := s.postJSON(ctx, "/v1/auth/cli/sessions", map[string]any{"baseUrl": strings.TrimSpace(oauthBaseURL)}, &resp, nil); err != nil {
		return nil, err
	}

	out := &CLILoginSession{
		SessionID:        strings.TrimSpace(resp.SessionID),
		SessionSecret:    strings.TrimSpace(resp.SessionSecret),
		VerificationPath: strings.TrimSpace(resp.VerificationPath),
		VerificationURL:  strings.TrimSpace(resp.VerificationURL),
		ExpiresAt:        derefTime(resp.ExpiresAt),
		PollInterval:     2 * time.Second,
	}
	if resp.PollIntervalSeconds > 0 {
		out.PollInterval = time.Duration(resp.PollIntervalSeconds) * time.Second
	}

	if out.VerificationURL == "" && out.VerificationPath != "" {
		base, err := url.Parse(strings.TrimSpace(oauthBaseURL))
		if err == nil {
			rel, relErr := url.Parse(out.VerificationPath)
			if relErr == nil {
				out.VerificationURL = base.ResolveReference(rel).String()
			}
		}
	}
	if out.SessionID == "" || out.SessionSecret == "" || out.VerificationURL == "" {
		return nil, errors.New("invalid create cli login session response")
	}
	return out, nil
}

func (s *Service) PollCLILoginSession(ctx context.Context, sessionID string, sessionSecret string) (*CLILoginSessionPoll, error) {
	sessionID = strings.TrimSpace(sessionID)
	sessionSecret = strings.TrimSpace(sessionSecret)
	if sessionID == "" || sessionSecret == "" {
		return nil, errors.New("session id and secret are required")
	}

	var resp struct {
		Status string `json:"status,omitempty"`
		Login  *struct {
			UserID               string     `json:"userId,omitempty"`
			Roles                []string   `json:"roles,omitempty"`
			AccessToken          string     `json:"accessToken,omitempty"`
			RefreshToken         string     `json:"refreshToken,omitempty"`
			AccessTokenExpiresAt *time.Time `json:"accessTokenExpiresAt,omitempty"`
		} `json:"login,omitempty"`
	}
	path := fmt.Sprintf("/v1/auth/cli/sessions/%s:poll", url.PathEscape(sessionID))
	if err := s.postJSON(ctx, path, map[string]any{"sessionSecret": sessionSecret}, &resp, nil); err != nil {
		return nil, err
	}

	out := &CLILoginSessionPoll{Status: strings.TrimSpace(resp.Status)}
	if resp.Login != nil {
		out.Login = &CLILoginSessionLogin{
			UserID:               strings.TrimSpace(resp.Login.UserID),
			Roles:                resp.Login.Roles,
			AccessToken:          strings.TrimSpace(resp.Login.AccessToken),
			RefreshToken:         strings.TrimSpace(resp.Login.RefreshToken),
			AccessTokenExpiresAt: resp.Login.AccessTokenExpiresAt,
		}
	}
	return out, nil
}

func (s *Service) CancelCLILoginSession(ctx context.Context, sessionID string, sessionSecret string) error {
	sessionID = strings.TrimSpace(sessionID)
	sessionSecret = strings.TrimSpace(sessionSecret)
	if sessionID == "" || sessionSecret == "" {
		return errors.New("session id and secret are required")
	}
	path := fmt.Sprintf("/v1/auth/cli/sessions/%s:cancel", url.PathEscape(sessionID))
	return s.postJSON(ctx, path, map[string]any{"sessionSecret": sessionSecret}, nil, nil)
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
		AccessToken          string     `json:"accessToken,omitempty"`
		RefreshToken         string     `json:"refreshToken,omitempty"`
		AccessTokenExpiresAt *time.Time `json:"accessTokenExpiresAt,omitempty"`
		Roles                []string   `json:"roles,omitempty"`
	}
	if err := s.postJSON(ctx, "/v1/auth/register", payload, &resp, nil); err != nil {
		return nil, err
	}
	return s.saveSessionFromLogin(ctx, loginResult{
		UserID:               resp.UserID,
		Roles:                resp.Roles,
		AccessToken:          resp.AccessToken,
		RefreshToken:         resp.RefreshToken,
		AccessTokenExpiresAt: resp.AccessTokenExpiresAt,
		Username:             strings.TrimSpace(in.Username),
	})
}

type loginResult struct {
	UserID               string
	Roles                []string
	AccessToken          string
	RefreshToken         string
	AccessTokenExpiresAt *time.Time
	Username             string
}

func (s *Service) SaveLoginResult(ctx context.Context, in *CLILoginSessionLogin) (*session.Session, error) {
	if in == nil {
		return nil, errors.New("login result must not be nil")
	}
	return s.saveSessionFromLogin(ctx, loginResult{
		UserID:               in.UserID,
		Roles:                in.Roles,
		AccessToken:          in.AccessToken,
		RefreshToken:         in.RefreshToken,
		AccessTokenExpiresAt: in.AccessTokenExpiresAt,
	})
}

func (s *Service) saveSessionFromLogin(ctx context.Context, in loginResult) (*session.Session, error) {
	if s.store == nil {
		return nil, errors.New("session store is not configured")
	}

	sess := &session.Session{
		BaseURL:              s.baseURL,
		UserID:               strings.TrimSpace(in.UserID),
		Username:             strings.TrimSpace(in.Username),
		Roles:                in.Roles,
		AccessToken:          strings.TrimSpace(in.AccessToken),
		RefreshToken:         strings.TrimSpace(in.RefreshToken),
		AccessTokenExpiresAt: derefTime(in.AccessTokenExpiresAt),
	}
	if err := s.store.Save(sess); err != nil {
		return nil, err
	}
	_, _ = s.EnsureUsername(ctx)
	return s.store.Load()
}

// GetMyProfile fetches the authenticated user's profile from the users service.
func (s *Service) GetMyProfile(ctx context.Context) (*clientv1.UserProfile, error) {
	if s.store == nil {
		return nil, errors.New("session store is not configured")
	}

	cred, err := s.credential(ctx)
	if err != nil {
		return nil, err
	}
	if cred == nil || strings.TrimSpace(cred.Value) == "" {
		return nil, errors.New("not logged in")
	}

	var resp struct {
		Profile *clientv1.UserProfile `json:"profile,omitempty"`
	}
	headers := make(http.Header)
	headers.Set(cred.Header, cred.Value)

	if err := s.getJSON(ctx, "/v1/me/profile", &resp, headers); err != nil {
		// Backwards-compat fallback for older servers that don't implement /v1/me/profile.
		if strings.Contains(err.Error(), "api error (404)") {
			sess, loadErr := s.store.Load()
			if loadErr != nil {
				return nil, loadErr
			}
			if sess == nil || strings.TrimSpace(sess.UserID) == "" {
				return nil, errors.New("not logged in")
			}
			c, clientErr := s.client(true)
			if clientErr != nil {
				return nil, clientErr
			}
			out, getErr := c.GetProfile(ctx, &clientv1.GetProfileRequest{UserID: sess.UserID})
			if getErr != nil {
				return nil, getErr
			}
			if out == nil {
				return nil, nil
			}
			return out.Profile, nil
		}
		return nil, err
	}
	return resp.Profile, nil
}

// EnsureUsername ensures the local session has a username, fetching it from the users service if needed.
func (s *Service) EnsureUsername(ctx context.Context) (string, error) {
	if s.store == nil {
		return "", errors.New("session store is not configured")
	}
	sess, err := s.store.Load()
	if err != nil {
		return "", err
	}
	if sess == nil {
		return "", errors.New("not logged in")
	}
	if username := strings.TrimSpace(sess.Username); username != "" {
		return username, nil
	}

	profile, err := s.GetMyProfile(ctx)
	if err != nil {
		return "", err
	}
	if profile == nil {
		return "", nil
	}
	username := strings.TrimSpace(profile.Username)
	if username == "" {
		return "", nil
	}
	sess.Username = username
	if err := s.store.Save(sess); err != nil {
		return "", err
	}
	return username, nil
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
	return s.refreshWithMode(ctx, false)
}

func (s *Service) refreshForced(ctx context.Context) error {
	return s.refreshWithMode(ctx, true)
}

func (s *Service) refreshWithMode(ctx context.Context, force bool) error {
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
	if !force && strings.TrimSpace(latest.AccessToken) != "" && !tokenExpiringSoon(latest.AccessTokenExpiresAt) {
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

type authRetryKey struct{}

type authRetryRoundTripper struct {
	base http.RoundTripper
	svc  *Service
}

func (t *authRetryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}

	resp, err := base.RoundTrip(req)
	if err != nil || resp == nil {
		return resp, err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	if req == nil || req.Context() == nil {
		return resp, nil
	}
	if req.Context().Value(authRetryKey{}) != nil {
		return resp, nil
	}

	if req.Header.Get("Authorization") == "" {
		return resp, nil
	}
	if t.svc == nil {
		return resp, nil
	}

	if req.Body != nil && req.GetBody == nil {
		return resp, nil
	}

	if req.URL != nil && strings.HasPrefix(req.URL.Path, "/v1/auth/token/refresh") {
		return resp, nil
	}

	_ = resp.Body.Close()

	if err := t.svc.refreshForced(req.Context()); err != nil {
		return nil, err
	}
	cred, err := t.svc.credential(req.Context())
	if err != nil {
		return nil, err
	}

	retryCtx := context.WithValue(req.Context(), authRetryKey{}, true)
	retryReq := req.Clone(retryCtx)
	if req.GetBody != nil {
		body, bodyErr := req.GetBody()
		if bodyErr != nil {
			return nil, bodyErr
		}
		retryReq.Body = body
	}
	retryReq.Header.Del("Authorization")
	if cred != nil && strings.TrimSpace(cred.Value) != "" {
		retryReq.Header.Set(cred.Header, cred.Value)
	}
	return base.RoundTrip(retryReq)
}

func (s *Service) getJSON(ctx context.Context, path string, out any, headers http.Header) error {
	return s.doJSON(ctx, http.MethodGet, path, nil, out, headers)
}

func (s *Service) postJSON(ctx context.Context, path string, payload any, out any, headers http.Header) error {
	return s.doJSON(ctx, http.MethodPost, path, payload, out, headers)
}

func (s *Service) doJSON(ctx context.Context, method string, path string, payload any, out any, headers http.Header) error {
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

	var body io.Reader
	var bodyBytes []byte
	if payload != nil {
		bodyBytes, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		body = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, base.ResolveReference(rel).String(), body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}
	req.Header.Set("Accept", "application/json")
	for k, vals := range headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}

	httpClient := &http.Client{
		Timeout: s.timeout,
		Transport: &authRetryRoundTripper{
			base: http.DefaultTransport,
			svc:  s,
		},
	}
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
