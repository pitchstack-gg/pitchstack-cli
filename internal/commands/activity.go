package commands

import (
	"context"
	"strings"

	"github.com/pitchstack-gg/pitchstack-cli/internal/pitchstack"

	"github.com/urfave/cli/v3"
)

func newActivityCommand() *cli.Command {
	return &cli.Command{
		Name:  "activity",
		Usage: "Browse activity",
		Commands: []*cli.Command{
			newActivityListCommand(),
		},
	}
}

func newActivityListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List activity feed items",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "page-size", Usage: "Page size"},
			&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
			&cli.StringSliceFlag{Name: "scope", Usage: "Activity scope (repeatable: following|shared|groups|system)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			req := &pitchstack.ListActivityFeedRequest{
				NextToken: strings.TrimSpace(cmd.String("next-token")),
			}
			if cmd.IsSet("page-size") && cmd.Int("page-size") > 0 {
				ps := int32(cmd.Int("page-size"))
				req.PageSize = &ps
			}

			for _, raw := range cmd.StringSlice("scope") {
				for _, part := range strings.Split(raw, ",") {
					scopeStr := strings.TrimSpace(part)
					if scopeStr == "" {
						continue
					}
					req.Scopes = append(req.Scopes, parseActivityScope(scopeStr))
				}
			}

			resp, err := st.Service.ListActivityFeed(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func parseActivityScope(v string) pitchstack.ActivityScope {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "unspecified":
		return pitchstack.ActivityScopeUnspecified
	case "following":
		return pitchstack.ActivityScopeFollowing
	case "shared":
		return pitchstack.ActivityScopeShared
	case "groups":
		return pitchstack.ActivityScopeGroups
	case "system":
		return pitchstack.ActivityScopeSystem
	default:
		upper := strings.ToUpper(strings.TrimSpace(v))
		if strings.HasPrefix(upper, "ACTIVITY_SCOPE_") {
			return pitchstack.ActivityScope(upper)
		}
		return pitchstack.ActivityScope(v)
	}
}
