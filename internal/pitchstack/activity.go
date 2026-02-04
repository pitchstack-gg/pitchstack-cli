package pitchstack

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type ActivityScope string

const (
	ActivityScopeUnspecified ActivityScope = "ACTIVITY_SCOPE_UNSPECIFIED"
	ActivityScopeFollowing   ActivityScope = "ACTIVITY_SCOPE_FOLLOWING"
	ActivityScopeShared      ActivityScope = "ACTIVITY_SCOPE_SHARED"
	ActivityScopeGroups      ActivityScope = "ACTIVITY_SCOPE_GROUPS"
	ActivityScopeSystem      ActivityScope = "ACTIVITY_SCOPE_SYSTEM"
)

type ListActivityFeedRequest struct {
	PageSize  *int32
	NextToken string
	Scopes    []ActivityScope
}

type ActivityItem struct {
	ActivityID string `json:"activityId,omitempty"`
	Kind       string `json:"kind,omitempty"`
	ActorID    string `json:"actorId,omitempty"`
	Verb       string `json:"verb,omitempty"`

	ResourceType string `json:"resourceType,omitempty"`
	ResourceID   string `json:"resourceId,omitempty"`

	TargetUserID  string `json:"targetUserId,omitempty"`
	TargetGroupID string `json:"targetGroupId,omitempty"`

	Summary  string         `json:"summary,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`

	Count json.RawMessage `json:"count,omitempty"`

	FirstOccurredAt string `json:"firstOccurredAt,omitempty"`
	LastOccurredAt  string `json:"lastOccurredAt,omitempty"`
	CreatedAt       string `json:"createdAt,omitempty"`
}

type ListActivityFeedResponse struct {
	Items     []ActivityItem `json:"items,omitempty"`
	NextToken string         `json:"nextToken,omitempty"`
}

func (s *Service) ListActivityFeed(ctx context.Context, request *ListActivityFeedRequest) (*ListActivityFeedResponse, error) {
	if request == nil {
		return nil, errors.New("request must not be nil")
	}

	cred, err := s.credential(ctx)
	if err != nil {
		return nil, err
	}
	if cred == nil || strings.TrimSpace(cred.Value) == "" {
		return nil, errors.New("not logged in")
	}

	query := url.Values{}
	if request.PageSize != nil && *request.PageSize > 0 {
		query.Set("pageSize", strconv.Itoa(int(*request.PageSize)))
	}
	if token := strings.TrimSpace(request.NextToken); token != "" {
		query.Set("nextToken", token)
	}
	for _, scope := range request.Scopes {
		v := strings.TrimSpace(string(scope))
		if v != "" {
			query.Add("scopes", v)
		}
	}

	path := "/v1/activity"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	var resp ListActivityFeedResponse
	headers := make(http.Header)
	headers.Set(cred.Header, cred.Value)
	if err := s.getJSON(ctx, path, &resp, headers); err != nil {
		return nil, err
	}
	return &resp, nil
}
