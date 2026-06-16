package commands

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/pitchstack-gg/pitchstack-cli/internal/cardsdb"
	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
	"github.com/urfave/cli/v3"
)

func newResourceLikeCommand(resourceName string, resourceType clientv1.LikeableResourceType) *cli.Command {
	return newSDKCommand("like", "Like a "+resourceName, []cli.Flag{
		&cli.StringFlag{Name: "id", Usage: resourceName + " ID"},
	}, true, func(cmd *cli.Command, req *clientv1.LikeResourceRequest) error {
		if req.Resource == nil {
			req.Resource = &clientv1.LikeableResourceRef{}
		}
		req.Resource.ResourceType = resourceType
		setStringFlag(cmd, "id", &req.Resource.ResourceID)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.LikeResourceRequest) (any, error) {
		return c.LikeResource(ctx, req)
	})
}

func newResourceUnlikeCommand(resourceName string, resourceType clientv1.LikeableResourceType) *cli.Command {
	return newSDKCommand("unlike", "Unlike a "+resourceName, []cli.Flag{
		&cli.StringFlag{Name: "id", Usage: resourceName + " ID"},
	}, true, func(cmd *cli.Command, req *clientv1.UnlikeResourceRequest) error {
		if req.Resource == nil {
			req.Resource = &clientv1.LikeableResourceRef{}
		}
		req.Resource.ResourceType = resourceType
		setStringFlag(cmd, "id", &req.Resource.ResourceID)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UnlikeResourceRequest) (any, error) {
		return c.UnlikeResource(ctx, req)
	})
}

func newResourceTrendingCommand(resourceName string, resourceType clientv1.TrackableResourceType) *cli.Command {
	flags := []cli.Flag{
		requestFileFlag(),
		&cli.StringFlag{Name: "window", Usage: "Window (24h|7d|30d or raw enum)"},
		&cli.IntFlag{Name: "page-size", Usage: "Page size"},
		&cli.StringFlag{Name: "next-token", Usage: "Next page token"},
		&cli.BoolFlag{Name: "raw", Usage: "Print the raw API response"},
	}
	if resourceType == clientv1.TrackableResourceTypeCard {
		flags = append(flags, localCardsFlags()...)
	}
	return &cli.Command{
		Name:  "trending",
		Usage: "List trending " + resourceName,
		Flags: flags,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			var req clientv1.ListTrendingResourcesRequest
			if err := readRequestFile(cmd, &req); err != nil {
				return err
			}
			req.ResourceType = resourceType
			if cmd.IsSet("window") {
				req.Window = parseTrendingWindow(cmd.String("window"))
			}
			if cmd.IsSet("page-size") {
				req.PageSize = int32(cmd.Int("page-size"))
			}
			setStringFlag(cmd, "next-token", &req.NextPageToken)

			var resp *clientv1.ListTrendingResourcesResponse
			if err := withSDKClientNoWrite(ctx, cmd, true, func(c *clientv1.Client) error {
				var err error
				resp, err = c.ListTrendingResources(ctx, &req)
				return err
			}); err != nil {
				return err
			}
			if cmd.Bool("raw") {
				return writeJSON(cmd.Writer, resp)
			}

			cardSummaries := map[string]cardsdb.CardSummary{}
			if resourceType == clientv1.TrackableResourceTypeCard {
				var err error
				cardSummaries, err = loadTrendingCardSummaries(ctx, cmd, resp)
				if err != nil {
					_, _ = fmt.Fprintf(cmd.ErrWriter, "warning: could not enrich trending cards from local card database: %s\n", err.Error())
				}
			}
			return writeJSON(cmd.Writer, formatTrendingResources(resourceName, resourceType, req.Window, resp, cardSummaries))
		},
	}
}

func parseTrendingWindow(v string) clientv1.TrendingWindow {
	switch v {
	case "24h":
		return clientv1.TrendingWindow24H
	case "7d":
		return clientv1.TrendingWindow7D
	case "30d":
		return clientv1.TrendingWindow30D
	default:
		return clientv1.TrendingWindow(v)
	}
}

type trendingResourcesOutput struct {
	ResourceType  string               `json:"resourceType"`
	Window        string               `json:"window,omitempty"`
	Items         []trendingItemOutput `json:"items"`
	NextPageToken string               `json:"nextPageToken,omitempty"`
}

type trendingItemOutput struct {
	Rank          int                 `json:"rank"`
	ResourceType  string              `json:"resourceType"`
	ResourceID    string              `json:"resourceId"`
	Card          *trendingCardOutput `json:"card,omitempty"`
	ViewCount     int64               `json:"viewCount"`
	Score         float64             `json:"score"`
	LastViewedAt  string              `json:"lastViewedAt,omitempty"`
	LastViewedAgo string              `json:"lastViewedAgo,omitempty"`
}

type trendingCardOutput struct {
	ID                string   `json:"id"`
	Name              string   `json:"name,omitempty"`
	Types             []string `json:"types,omitempty"`
	Keywords          []string `json:"keywords,omitempty"`
	Cost              string   `json:"cost,omitempty"`
	Pitch             string   `json:"pitch,omitempty"`
	Power             string   `json:"power,omitempty"`
	Defense           string   `json:"defense,omitempty"`
	Health            string   `json:"health,omitempty"`
	Intelligence      string   `json:"intelligence,omitempty"`
	Arcane            string   `json:"arcane,omitempty"`
	ColorIdentity     string   `json:"colorIdentity,omitempty"`
	IsDoubleFacedCard bool     `json:"isDoubleFacedCard,omitempty"`
	DefaultImageURL   string   `json:"defaultImageUrl,omitempty"`
}

func withSDKClientNoWrite(ctx context.Context, cmd *cli.Command, authenticated bool, fn func(*clientv1.Client) error) error {
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
	return fn(c)
}

func loadTrendingCardSummaries(ctx context.Context, cmd *cli.Command, resp *clientv1.ListTrendingResourcesResponse) (map[string]cardsdb.CardSummary, error) {
	if resp == nil || len(resp.Resources) == 0 {
		return map[string]cardsdb.CardSummary{}, nil
	}
	ids := make([]string, 0, len(resp.Resources))
	seen := map[string]bool{}
	for _, item := range resp.Resources {
		if item.Resource == nil {
			continue
		}
		id := strings.TrimSpace(item.Resource.ResourceID)
		if id != "" && !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	if len(ids) == 0 {
		return map[string]cardsdb.CardSummary{}, nil
	}
	repo, _, _, err := openLocalCardsRepo(ctx, cmd)
	if err != nil {
		return nil, err
	}
	defer repo.Close()
	cards, err := repo.BatchGetCardSummaries(ctx, ids)
	if err != nil {
		return nil, err
	}
	return cards.Cards, nil
}

func formatTrendingResources(resourceName string, resourceType clientv1.TrackableResourceType, window clientv1.TrendingWindow, resp *clientv1.ListTrendingResourcesResponse, cards map[string]cardsdb.CardSummary) trendingResourcesOutput {
	out := trendingResourcesOutput{
		ResourceType: displayTrackableResourceType(resourceName, resourceType),
		Window:       displayTrendingWindow(window),
		Items:        []trendingItemOutput{},
	}
	if resp == nil {
		return out
	}
	out.NextPageToken = resp.NextPageToken
	for idx, item := range resp.Resources {
		itemResourceType := displayTrackableResourceType(resourceName, resourceType)
		resourceID := ""
		if item.Resource != nil {
			itemResourceType = displayTrackableResourceType(resourceName, item.Resource.ResourceType)
			resourceID = strings.TrimSpace(item.Resource.ResourceID)
		}
		row := trendingItemOutput{
			Rank:         idx + 1,
			ResourceType: itemResourceType,
			ResourceID:   resourceID,
			ViewCount:    item.ViewCount,
			Score:        roundFloat(item.Score, 4),
		}
		if item.LastViewedAt != nil && !item.LastViewedAt.IsZero() {
			row.LastViewedAt = item.LastViewedAt.UTC().Format(time.RFC3339)
			row.LastViewedAgo = formatRelativeTime(time.Since(*item.LastViewedAt))
		}
		if card, ok := cards[resourceID]; ok {
			row.Card = &trendingCardOutput{
				ID:                card.Identifier,
				Name:              card.Name,
				Types:             card.Types,
				Keywords:          card.Keywords,
				Cost:              card.Cost,
				Pitch:             card.Pitch,
				Power:             card.Power,
				Defense:           card.Defense,
				Health:            card.Health,
				Intelligence:      card.Intelligence,
				Arcane:            card.Arcane,
				ColorIdentity:     card.ColorIdentity,
				IsDoubleFacedCard: card.IsDoubleFacedCard,
				DefaultImageURL:   card.DefaultImageURL,
			}
		}
		out.Items = append(out.Items, row)
	}
	return out
}

func displayTrackableResourceType(resourceName string, resourceType clientv1.TrackableResourceType) string {
	switch resourceType {
	case clientv1.TrackableResourceTypeCard:
		return "card"
	case clientv1.TrackableResourceTypeDeck:
		return "deck"
	case clientv1.TrackableResourceTypeCollection:
		return "collection"
	case clientv1.TrackableResourceTypeUserProfile:
		return "userProfile"
	default:
		resourceType := strings.TrimSpace(string(resourceType))
		if resourceType == "" || resourceType == string(clientv1.TrackableResourceTypeUnspecified) {
			return strings.TrimSuffix(resourceName, "s")
		}
		resourceType = strings.TrimPrefix(resourceType, "TRACKABLE_RESOURCE_TYPE_")
		resourceType = strings.ToLower(resourceType)
		return strings.ReplaceAll(resourceType, "_", "-")
	}
}

func displayTrendingWindow(window clientv1.TrendingWindow) string {
	switch window {
	case clientv1.TrendingWindow24H, "":
		return "24h"
	case clientv1.TrendingWindow7D:
		return "7d"
	case clientv1.TrendingWindow30D:
		return "30d"
	default:
		return strings.TrimSpace(string(window))
	}
}

func roundFloat(v float64, places int) float64 {
	if places < 0 {
		return v
	}
	scale := math.Pow10(places)
	return math.Round(v*scale) / scale
}

func formatRelativeTime(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/(24*365)))
	}
}
