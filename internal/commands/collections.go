package commands

import (
	"context"
	"strings"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"

	"github.com/urfave/cli/v3"
)

func newCollectionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "collections",
		Usage: "Manage collections",
		Commands: []*cli.Command{
			newCollectionsListCommand(),
			newCollectionsGetCommand(),
			newCollectionsCreateCommand(),
			newCollectionsUpdateCommand(),
			newCollectionsDeleteCommand(),
			newCollectionsPermissionsCommand(),
			newCollectionItemsCommand(),
		},
	}
}

func newCollectionsPermissionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "permissions",
		Usage: "Manage collection access permissions",
		Commands: []*cli.Command{
			newCollectionsPermissionsGrantCommand(),
			newCollectionsPermissionsRevokeCommand(),
		},
	}
}

func newCollectionsPermissionsGrantCommand() *cli.Command {
	return &cli.Command{
		Name:  "grant",
		Usage: "Grant a user access to a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "collection-id", Usage: "Collection ID", Required: true},
			&cli.StringFlag{Name: "subject-id", Usage: "User ID to grant access to", Required: true},
			&cli.StringFlag{Name: "permission", Usage: "Permission (reader|writer)", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			perm := parseCollectionPermission(cmd.String("permission"))
			if perm == clientv1.CollectionPermissionUnspecified {
				return cli.Exit("--permission must be reader|writer", 2)
			}

			collectionID := strings.TrimSpace(cmd.String("collection-id"))
			subjectID := strings.TrimSpace(cmd.String("subject-id"))

			if _, err := st.Service.GrantCollectionAccess(ctx, &clientv1.GrantCollectionAccessRequest{
				CollectionID: collectionID,
				SubjectID:    subjectID,
				Permission:   perm,
			}); err != nil {
				return err
			}

			return writeJSON(cmd.Writer, map[string]any{
				"collectionId": collectionID,
				"subjectId":    subjectID,
				"permission":   perm,
				"granted":      true,
			})
		},
	}
}

func newCollectionsPermissionsRevokeCommand() *cli.Command {
	return &cli.Command{
		Name:  "revoke",
		Usage: "Revoke a user's access to a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "collection-id", Usage: "Collection ID", Required: true},
			&cli.StringFlag{Name: "subject-id", Usage: "User ID to revoke access from", Required: true},
			&cli.StringFlag{Name: "permission", Usage: "Permission (reader|writer)", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			perm := parseCollectionPermission(cmd.String("permission"))
			if perm == clientv1.CollectionPermissionUnspecified {
				return cli.Exit("--permission must be reader|writer", 2)
			}

			collectionID := strings.TrimSpace(cmd.String("collection-id"))
			subjectID := strings.TrimSpace(cmd.String("subject-id"))

			if _, err := st.Service.RevokeCollectionAccess(ctx, &clientv1.RevokeCollectionAccessRequest{
				CollectionID: collectionID,
				SubjectID:    subjectID,
				Permission:   perm,
			}); err != nil {
				return err
			}

			return writeJSON(cmd.Writer, map[string]any{
				"collectionId": collectionID,
				"subjectId":    subjectID,
				"permission":   perm,
				"revoked":      true,
			})
		},
	}
}

func newCollectionsListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List collections",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "scope", Usage: "Scope (owned|shared|accessible)"},
			&cli.StringFlag{Name: "user-id", Usage: "Filter by user ID"},
			&cli.IntFlag{Name: "page-size", Usage: "Page size"},
			&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			req := &clientv1.ListCollectionsRequest{
				Scope:     parseCollectionScope(cmd.String("scope")),
				UserID:    strings.TrimSpace(cmd.String("user-id")),
				NextToken: strings.TrimSpace(cmd.String("next-token")),
			}
			if cmd.IsSet("page-size") && cmd.Int("page-size") > 0 {
				ps := int32(cmd.Int("page-size"))
				req.PageSize = &ps
			}

			resp, err := st.Service.ListCollections(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCollectionsGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Collection ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			id := strings.TrimSpace(cmd.String("id"))
			resp, err := st.Service.GetCollection(ctx, &clientv1.GetCollectionRequest{CollectionID: id})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCollectionsCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Usage: "Collection name", Required: true},
			&cli.StringFlag{Name: "type", Usage: "Collection type (binder|wantlist|tradelist|list)", Required: true},
			&cli.StringFlag{Name: "description", Usage: "Collection description"},
			&cli.StringFlag{Name: "visibility", Usage: "Visibility (private|shared|public)", Value: "private"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			collectionType := parseCollectionType(cmd.String("type"))
			if collectionType == clientv1.CollectionTypeUnspecified {
				return cli.Exit("--type must be binder|wantlist|tradelist|list", 2)
			}

			req := &clientv1.CreateCollectionRequest{
				Name:           strings.TrimSpace(cmd.String("name")),
				CollectionType: collectionType,
				Description:    strings.TrimSpace(cmd.String("description")),
				Visibility:     parseVisibility(cmd.String("visibility")),
			}

			resp, err := st.Service.CreateCollection(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCollectionsUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Collection ID", Required: true},
			&cli.StringFlag{Name: "name", Usage: "New name"},
			&cli.StringFlag{Name: "description", Usage: "New description"},
			&cli.StringFlag{Name: "visibility", Usage: "New visibility (private|shared|public)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			id := strings.TrimSpace(cmd.String("id"))
			var anyUpdate bool

			var updateReq clientv1.UpdateCollectionRequest
			updateReq.CollectionID = id
			if cmd.IsSet("name") {
				v := strings.TrimSpace(cmd.String("name"))
				updateReq.Name = &v
				anyUpdate = true
			}
			if cmd.IsSet("description") {
				v := strings.TrimSpace(cmd.String("description"))
				updateReq.Description = &v
				anyUpdate = true
			}

			if anyUpdate {
				if _, err := st.Service.UpdateCollection(ctx, &updateReq); err != nil {
					return err
				}
			}

			if cmd.IsSet("visibility") {
				v := parseVisibility(cmd.String("visibility"))
				if v == clientv1.VisibilityLevelUnspecified {
					return cli.Exit("--visibility must be private|shared|public", 2)
				}
				if _, err := st.Service.UpdateCollectionVisibility(ctx, &clientv1.UpdateCollectionVisibilityRequest{
					CollectionID: id,
					Visibility:   &v,
				}); err != nil {
					return err
				}
				anyUpdate = true
			}

			if !anyUpdate {
				return cli.Exit("no updates provided", 2)
			}

			resp, err := st.Service.GetCollection(ctx, &clientv1.GetCollectionRequest{CollectionID: id})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCollectionsDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Collection ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			id := strings.TrimSpace(cmd.String("id"))
			if _, err := st.Service.DeleteCollection(ctx, &clientv1.DeleteCollectionRequest{CollectionID: id}); err != nil {
				return err
			}
			return writeJSON(cmd.Writer, map[string]any{"deleted": id})
		},
	}
}

func parseCollectionScope(v string) clientv1.CollectionListScope {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "unspecified":
		return clientv1.CollectionListScopeUnspecified
	case "owned":
		return clientv1.CollectionListScopeOwned
	case "shared":
		return clientv1.CollectionListScopeShared
	case "accessible":
		return clientv1.CollectionListScopeAccessible
	default:
		return clientv1.CollectionListScope(v)
	}
}

func parseCollectionType(v string) clientv1.CollectionType {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "unspecified":
		return clientv1.CollectionTypeUnspecified
	case "binder":
		return clientv1.CollectionTypeBinder
	case "wantlist":
		return clientv1.CollectionTypeWantlist
	case "tradelist":
		return clientv1.CollectionTypeTradelist
	case "list":
		return clientv1.CollectionTypeList
	default:
		return clientv1.CollectionType(v)
	}
}

func parseCollectionPermission(v string) clientv1.CollectionPermission {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "unspecified":
		return clientv1.CollectionPermissionUnspecified
	case "reader", "read", "r":
		return clientv1.CollectionPermissionReader
	case "writer", "write", "w":
		return clientv1.CollectionPermissionWriter
	default:
		return clientv1.CollectionPermission(v)
	}
}

func parseVisibility(v string) clientv1.VisibilityLevel {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "unspecified":
		return clientv1.VisibilityLevelUnspecified
	case "private":
		return clientv1.VisibilityLevelPrivate
	case "shared":
		return clientv1.VisibilityLevelShared
	case "public":
		return clientv1.VisibilityLevelPublic
	default:
		return clientv1.VisibilityLevel(v)
	}
}

func newCollectionItemsCommand() *cli.Command {
	return &cli.Command{
		Name:  "items",
		Usage: "Manage collection items",
		Commands: []*cli.Command{
			newCollectionItemsListCommand(),
			newCollectionItemsGetCommand(),
			newCollectionItemsAddCommand(),
			newCollectionItemsUpdateCommand(),
			newCollectionItemsDeleteCommand(),
		},
	}
}

func newCollectionItemsListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List collection items",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "collection-id", Usage: "Collection ID", Required: true},
			&cli.StringFlag{Name: "card-id", Usage: "Filter by card ID"},
			&cli.StringFlag{Name: "printing-id", Usage: "Filter by printing ID"},
			&cli.StringFlag{Name: "product-id", Usage: "Filter by product ID"},
			&cli.IntFlag{Name: "page-size", Usage: "Page size"},
			&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			req := &clientv1.ListCollectionItemsRequest{
				CollectionID: strings.TrimSpace(cmd.String("collection-id")),
				CardID:       strings.TrimSpace(cmd.String("card-id")),
				PrintingID:   strings.TrimSpace(cmd.String("printing-id")),
				ProductID:    strings.TrimSpace(cmd.String("product-id")),
				NextToken:    strings.TrimSpace(cmd.String("next-token")),
			}
			if cmd.IsSet("page-size") && cmd.Int("page-size") > 0 {
				ps := int32(cmd.Int("page-size"))
				req.PageSize = &ps
			}
			resp, err := st.Service.ListCollectionItems(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCollectionItemsGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a collection item",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Item ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			id := strings.TrimSpace(cmd.String("id"))
			resp, err := st.Service.GetCollectionItem(ctx, &clientv1.GetCollectionItemRequest{ItemID: id})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCollectionItemsAddCommand() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "Add an item to a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "collection-id", Usage: "Collection ID", Required: true},
			&cli.StringFlag{Name: "product-id", Usage: "Product ID", Required: true},
			&cli.IntFlag{Name: "quantity", Usage: "Quantity (>= 1)", Value: 1},
			&cli.StringFlag{Name: "condition", Usage: "Condition (near_mint|lightly_played|heavily_played|damaged)", Value: "near_mint"},
			&cli.Float64Flag{Name: "value", Usage: "Optional value"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			qty := cmd.Int("quantity")
			if qty < 1 {
				return cli.Exit("--quantity must be >= 1", 2)
			}
			cond, ok := parseCondition(cmd.String("condition"))
			if !ok {
				return cli.Exit("--condition must be near_mint|lightly_played|heavily_played|damaged", 2)
			}

			req := &clientv1.CreateCollectionItemRequest{
				CollectionID: strings.TrimSpace(cmd.String("collection-id")),
				ProductID:    strings.TrimSpace(cmd.String("product-id")),
				Quantity:     int32(qty),
				Condition:    cond,
				Value:        float64Ptr(cmd, "value"),
			}

			resp, err := st.Service.CreateCollectionItem(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCollectionItemsUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a collection item",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Item ID", Required: true},
			&cli.IntFlag{Name: "quantity", Usage: "Quantity (>= 1)"},
			&cli.StringFlag{Name: "condition", Usage: "Condition (near_mint|lightly_played|heavily_played|damaged)"},
			&cli.Float64Flag{Name: "value", Usage: "Value"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			req := &clientv1.UpdateCollectionItemRequest{
				ItemID: strings.TrimSpace(cmd.String("id")),
			}
			var anyUpdate bool

			if cmd.IsSet("quantity") {
				qty := cmd.Int("quantity")
				if qty < 1 {
					return cli.Exit("--quantity must be >= 1", 2)
				}
				q := int32(qty)
				req.Quantity = &q
				anyUpdate = true
			}
			if cmd.IsSet("condition") {
				cond, ok := parseCondition(cmd.String("condition"))
				if !ok {
					return cli.Exit("--condition must be near_mint|lightly_played|heavily_played|damaged", 2)
				}
				req.Condition = &cond
				anyUpdate = true
			}
			if cmd.IsSet("value") {
				req.Value = float64Ptr(cmd, "value")
				anyUpdate = true
			}

			if !anyUpdate {
				return cli.Exit("no updates provided", 2)
			}

			resp, err := st.Service.UpdateCollectionItem(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCollectionItemsDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a collection item",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Item ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			id := strings.TrimSpace(cmd.String("id"))
			if _, err := st.Service.DeleteCollectionItem(ctx, &clientv1.DeleteCollectionItemRequest{ItemID: id}); err != nil {
				return err
			}
			return writeJSON(cmd.Writer, map[string]any{"deleted": id})
		},
	}
}

func parseCondition(v string) (clientv1.Condition, bool) {
	n := strings.ToLower(strings.TrimSpace(v))
	n = strings.ReplaceAll(n, "-", "_")
	switch n {
	case "", "unspecified":
		return clientv1.ConditionUnspecified, false
	case "near_mint", "nm":
		return clientv1.ConditionNearMint, true
	case "lightly_played", "lp":
		return clientv1.ConditionLightlyPlayed, true
	case "heavily_played", "hp":
		return clientv1.ConditionHeavilyPlayed, true
	case "damaged", "dmg":
		return clientv1.ConditionDamaged, true
	default:
		return clientv1.Condition(v), true
	}
}
