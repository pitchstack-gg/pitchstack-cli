package commands

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
			newCollectionsCountsCommand(),
			newCollectionsGetCommand(),
			newCollectionsHistoryCommand(),
			newCollectionsBatchGetCommand(),
			newCollectionsCreateCommand(),
			newCollectionsUpdateCommand(),
			newCollectionsArtCommand(),
			newCollectionsDeleteCommand(),
			newCollectionsExportCommand(),
			newCollectionsImportCommand(),
			newCollectionsValuationCommand(),
			newCollectionsTradeItemsCommand(),
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
			newCollectionsPermissionsGetCommand(),
			newCollectionsPermissionsListCommand(),
			newCollectionsPermissionsGrantCommand(),
			newCollectionsPermissionsRevokeCommand(),
			newCollectionsPermissionsStopShareCommand(),
		},
	}
}

func newCollectionsPermissionsGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get your effective permission for a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "collection-id", Usage: "Collection ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			resp, err := st.Service.GetCollectionAccess(ctx, &clientv1.GetCollectionAccessRequest{
				CollectionID: strings.TrimSpace(cmd.String("collection-id")),
			})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newCollectionsPermissionsListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List explicit access grants for a collection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "collection-id", Usage: "Collection ID", Required: true},
			&cli.IntFlag{Name: "page-size", Usage: "Page size"},
			&cli.StringFlag{Name: "next-token", Usage: "Pagination token"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			req := &clientv1.ListCollectionAccessGrantsRequest{
				CollectionID: strings.TrimSpace(cmd.String("collection-id")),
				NextToken:    strings.TrimSpace(cmd.String("next-token")),
			}
			if cmd.IsSet("page-size") && cmd.Int("page-size") > 0 {
				ps := int32(cmd.Int("page-size"))
				req.PageSize = &ps
			}

			resp, err := st.Service.ListCollectionAccessGrants(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
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

func newCollectionsPermissionsStopShareCommand() *cli.Command {
	return newSDKCommand("stop-share", "Remove all explicit collection shares", []cli.Flag{&cli.StringFlag{Name: "collection-id", Usage: "Collection ID"}}, true, func(cmd *cli.Command, req *clientv1.StopCollectionShareRequest) error {
		setStringFlag(cmd, "collection-id", &req.CollectionID)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.StopCollectionShareRequest) (any, error) {
		return c.StopCollectionShare(ctx, req)
	})
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
			query := make(url.Values)
			if scope := parseCollectionScope(cmd.String("scope")); scope != clientv1.CollectionListScopeUnspecified {
				query.Set("scope", string(scope))
			}
			setQueryString(query, "userId", cmd.String("user-id"))
			if cmd.IsSet("page-size") && cmd.Int("page-size") > 0 {
				query.Set("pageSize", fmt.Sprintf("%d", cmd.Int("page-size")))
			}
			setQueryString(query, "nextToken", cmd.String("next-token"))
			return writeAuthenticatedJSON(ctx, cmd, http.MethodGet, pathWithQuery("/v1/collections", query), nil)
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
			id := strings.TrimSpace(cmd.String("id"))
			return writeAuthenticatedJSON(ctx, cmd, http.MethodGet, "/v1/collections/"+url.PathEscape(id), nil)
		},
	}
}

func newCollectionsHistoryCommand() *cli.Command {
	return &cli.Command{
		Name:  "history",
		Usage: "Get collection history",
		Flags: []cli.Flag{&cli.StringFlag{Name: "id", Usage: "Collection ID", Required: true}},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			path := "/v1/collections/" + url.PathEscape(strings.TrimSpace(cmd.String("id"))) + "/history"
			return writeAuthenticatedJSON(ctx, cmd, http.MethodGet, path, nil)
		},
	}
}

func newCollectionsBatchGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "batch-get",
		Usage: "Batch get collections",
		Flags: []cli.Flag{
			requestFileFlag(),
			repeatedIDsFlag("id", "Collection ID (repeatable or comma-separated)"),
			&cli.BoolFlag{Name: "allow-partial", Usage: "Allow partial results"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			payload, err := readObjectPayload(cmd)
			if err != nil {
				return err
			}
			if cmd.IsSet("id") {
				payload["collectionIds"] = splitCSV(cmd.StringSlice("id"))
			}
			setPayloadBoolFlag(cmd, "allow-partial", "allowPartial", payload)
			return writeAuthenticatedJSON(ctx, cmd, http.MethodPost, "/v1/collections:batchGet", payload)
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
			collectionType := parseCollectionType(cmd.String("type"))
			if collectionType == clientv1.CollectionTypeUnspecified {
				return cli.Exit("--type must be binder|wantlist|tradelist|list", 2)
			}

			payload := map[string]any{
				"name":           strings.TrimSpace(cmd.String("name")),
				"collectionType": string(collectionType),
			}
			setPayloadStringFlag(cmd, "description", "description", payload)
			if visibility := parseVisibility(cmd.String("visibility")); visibility != clientv1.VisibilityLevelUnspecified {
				payload["visibility"] = string(visibility)
			}
			return writeAuthenticatedJSON(ctx, cmd, http.MethodPost, "/v1/collections", payload)
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
			var last map[string]any
			var anyUpdate bool

			payload := map[string]any{}
			if cmd.IsSet("name") {
				payload["name"] = strings.TrimSpace(cmd.String("name"))
				anyUpdate = true
			}
			if cmd.IsSet("description") {
				payload["description"] = strings.TrimSpace(cmd.String("description"))
				anyUpdate = true
			}

			if anyUpdate {
				if err := st.Service.DoJSON(ctx, http.MethodPut, "/v1/collections/"+url.PathEscape(id), payload, &last, true); err != nil {
					return err
				}
			}

			if cmd.IsSet("visibility") {
				v := parseVisibility(cmd.String("visibility"))
				if v == clientv1.VisibilityLevelUnspecified {
					return cli.Exit("--visibility must be private|shared|public", 2)
				}
				if err := st.Service.DoJSON(ctx, http.MethodPatch, "/v1/collections/"+url.PathEscape(id)+"/visibility", map[string]any{"visibility": string(v)}, &last, true); err != nil {
					return err
				}
				anyUpdate = true
			}

			if !anyUpdate {
				return cli.Exit("no updates provided", 2)
			}

			if last != nil {
				return writeJSON(cmd.Writer, last)
			}
			return writeAuthenticatedJSON(ctx, cmd, http.MethodGet, "/v1/collections/"+url.PathEscape(id), nil)
		},
	}
}

func newCollectionsArtCommand() *cli.Command {
	return &cli.Command{
		Name:  "art",
		Usage: "Update collection artwork selection",
		Flags: []cli.Flag{
			requestFileFlag(),
			&cli.StringFlag{Name: "id", Usage: "Collection ID", Required: true},
			&cli.StringFlag{Name: "selected-art-printing-id", Usage: "Selected art printing ID"},
			&cli.BoolFlag{Name: "clear-selected-art", Usage: "Clear selected artwork"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			payload, err := readObjectPayload(cmd)
			if err != nil {
				return err
			}
			setPayloadStringFlag(cmd, "selected-art-printing-id", "selectedArtPrintingId", payload)
			setPayloadBoolFlag(cmd, "clear-selected-art", "clearSelectedArt", payload)
			path := "/v1/collections/" + url.PathEscape(strings.TrimSpace(cmd.String("id"))) + "/art"
			return writeAuthenticatedJSON(ctx, cmd, http.MethodPut, path, payload)
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

func newCollectionsExportCommand() *cli.Command {
	return &cli.Command{
		Name:  "export",
		Usage: "Export a collection",
		Flags: append(pageFlags(), &cli.StringFlag{Name: "id", Usage: "Collection ID", Required: true}),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			query := make(url.Values)
			if cmd.IsSet("page-size") && cmd.Int("page-size") > 0 {
				query.Set("pageSize", fmt.Sprintf("%d", cmd.Int("page-size")))
			}
			setQueryString(query, "nextToken", cmd.String("next-token"))
			path := "/v1/collections/" + url.PathEscape(strings.TrimSpace(cmd.String("id"))) + ":export"
			return writeAuthenticatedJSON(ctx, cmd, http.MethodGet, pathWithQuery(path, query), nil)
		},
	}
}

func newCollectionsImportCommand() *cli.Command {
	return &cli.Command{
		Name:  "import",
		Usage: "Import a collection from JSON",
		Flags: []cli.Flag{
			requestFileFlag(),
			&cli.StringFlag{Name: "id", Usage: "Collection ID"},
			&cli.StringFlag{Name: "name", Usage: "Collection name"},
			&cli.StringFlag{Name: "type", Usage: "Collection type"},
			&cli.StringFlag{Name: "description", Usage: "Description"},
			&cli.StringFlag{Name: "visibility", Usage: "Visibility"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			payload, err := readObjectPayload(cmd)
			if err != nil {
				return err
			}
			setPayloadStringFlag(cmd, "id", "collectionId", payload)
			setPayloadStringFlag(cmd, "name", "name", payload)
			if cmd.IsSet("type") {
				payload["collectionType"] = string(parseCollectionType(cmd.String("type")))
			}
			setPayloadStringFlag(cmd, "description", "description", payload)
			if cmd.IsSet("visibility") {
				payload["visibility"] = string(parseVisibility(cmd.String("visibility")))
			}
			return writeAuthenticatedJSON(ctx, cmd, http.MethodPost, "/v1/collections:import", payload)
		},
	}
}

func newCollectionsValuationCommand() *cli.Command {
	return newSDKCommand("valuation", "Get collection valuation", []cli.Flag{
		&cli.StringFlag{Name: "id", Usage: "Collection ID"},
		&cli.StringFlag{Name: "source", Usage: "Price source"},
	}, true, func(cmd *cli.Command, req *clientv1.GetCollectionValuationRequest) error {
		setStringFlag(cmd, "id", &req.CollectionID)
		setStringFlag(cmd, "source", &req.Source)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetCollectionValuationRequest) (any, error) {
		return c.GetCollectionValuation(ctx, req)
	})
}

func newCollectionsTradeItemsCommand() *cli.Command {
	return &cli.Command{
		Name:  "trade-items",
		Usage: "List tradable collection items",
		Flags: append(pageFlags(), &cli.StringFlag{Name: "user-id", Usage: "Filter by user ID"}),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			query := make(url.Values)
			setQueryString(query, "userId", cmd.String("user-id"))
			if cmd.IsSet("page-size") && cmd.Int("page-size") > 0 {
				query.Set("pageSize", fmt.Sprintf("%d", cmd.Int("page-size")))
			}
			setQueryString(query, "nextToken", cmd.String("next-token"))
			return writeAuthenticatedJSON(ctx, cmd, http.MethodGet, pathWithQuery("/v1/trade_items", query), nil)
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
			newCollectionItemsAdjustCommand(),
			newCollectionItemsTransferCommand(),
			newCollectionItemsBatchGetCommand(),
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
			query := make(url.Values)
			setQueryString(query, "collectionId", cmd.String("collection-id"))
			setQueryString(query, "cardId", cmd.String("card-id"))
			setQueryString(query, "printingId", cmd.String("printing-id"))
			setQueryString(query, "productId", cmd.String("product-id"))
			if cmd.IsSet("page-size") && cmd.Int("page-size") > 0 {
				query.Set("pageSize", fmt.Sprintf("%d", cmd.Int("page-size")))
			}
			setQueryString(query, "nextToken", cmd.String("next-token"))
			return writeAuthenticatedJSON(ctx, cmd, http.MethodGet, pathWithQuery("/v1/collection_items", query), nil)
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
			id := strings.TrimSpace(cmd.String("id"))
			return writeAuthenticatedJSON(ctx, cmd, http.MethodGet, "/v1/collection_items/"+url.PathEscape(id), nil)
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
			&cli.StringFlag{Name: "item-id", Usage: "Optional item ID (idempotency)"},
			&cli.IntFlag{Name: "trade-quantity", Usage: "Trade quantity"},
			&cli.StringFlag{Name: "notes", Usage: "Notes"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			qty := cmd.Int("quantity")
			if qty < 1 {
				return cli.Exit("--quantity must be >= 1", 2)
			}
			cond, ok := parseCondition(cmd.String("condition"))
			if !ok {
				return cli.Exit("--condition must be near_mint|lightly_played|heavily_played|damaged", 2)
			}

			payload := map[string]any{
				"collectionId": strings.TrimSpace(cmd.String("collection-id")),
				"productId":    strings.TrimSpace(cmd.String("product-id")),
				"quantity":     qty,
				"condition":    string(cond),
			}
			if cmd.IsSet("value") {
				payload["value"] = cmd.Float64("value")
			}
			setPayloadStringFlag(cmd, "item-id", "itemId", payload)
			setPayloadIntFlag(cmd, "trade-quantity", "tradeQuantity", payload)
			setPayloadStringFlag(cmd, "notes", "notes", payload)
			return writeAuthenticatedJSON(ctx, cmd, http.MethodPost, "/v1/collection_items", payload)
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
			&cli.BoolFlag{Name: "pinned", Usage: "Pin or unpin item"},
			&cli.StringFlag{Name: "expected-updated-at", Usage: "Expected updated_at precondition (RFC3339)"},
			&cli.StringFlag{Name: "client-mutation-id", Usage: "Client mutation ID"},
			&cli.IntFlag{Name: "trade-quantity", Usage: "Trade quantity"},
			&cli.StringFlag{Name: "notes", Usage: "Notes"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			payload := map[string]any{}
			var anyUpdate bool

			if cmd.IsSet("quantity") {
				qty := cmd.Int("quantity")
				if qty < 1 {
					return cli.Exit("--quantity must be >= 1", 2)
				}
				payload["quantity"] = qty
				anyUpdate = true
			}
			if cmd.IsSet("condition") {
				cond, ok := parseCondition(cmd.String("condition"))
				if !ok {
					return cli.Exit("--condition must be near_mint|lightly_played|heavily_played|damaged", 2)
				}
				payload["condition"] = string(cond)
				anyUpdate = true
			}
			if cmd.IsSet("value") {
				payload["value"] = cmd.Float64("value")
				anyUpdate = true
			}
			if cmd.IsSet("pinned") {
				payload["pinned"] = cmd.Bool("pinned")
				anyUpdate = true
			}
			if cmd.IsSet("expected-updated-at") {
				payload["expectedUpdatedAt"] = strings.TrimSpace(cmd.String("expected-updated-at"))
				anyUpdate = true
			}
			if cmd.IsSet("client-mutation-id") {
				payload["clientMutationId"] = strings.TrimSpace(cmd.String("client-mutation-id"))
				anyUpdate = true
			}
			if cmd.IsSet("trade-quantity") {
				payload["tradeQuantity"] = cmd.Int("trade-quantity")
				anyUpdate = true
			}
			if cmd.IsSet("notes") {
				payload["notes"] = strings.TrimSpace(cmd.String("notes"))
				anyUpdate = true
			}

			if !anyUpdate {
				return cli.Exit("no updates provided", 2)
			}

			path := "/v1/collection_items/" + url.PathEscape(strings.TrimSpace(cmd.String("id")))
			return writeAuthenticatedJSON(ctx, cmd, http.MethodPut, path, payload)
		},
	}
}

func newCollectionItemsAdjustCommand() *cli.Command {
	return newSDKCommand("adjust", "Adjust collection item quantity", []cli.Flag{
		&cli.StringFlag{Name: "collection-id", Usage: "Collection ID"},
		&cli.StringFlag{Name: "product-id", Usage: "Product ID"},
		&cli.StringFlag{Name: "condition", Usage: "Condition"},
		&cli.IntFlag{Name: "quantity-delta", Usage: "Quantity delta"},
		&cli.StringFlag{Name: "item-id", Usage: "Item ID"},
		&cli.StringFlag{Name: "client-mutation-id", Usage: "Client mutation ID"},
	}, true, func(cmd *cli.Command, req *clientv1.AdjustCollectionItemQuantityRequest) error {
		setStringFlag(cmd, "collection-id", &req.CollectionID)
		setStringFlag(cmd, "product-id", &req.ProductID)
		if cmd.IsSet("condition") {
			cond, ok := parseCondition(cmd.String("condition"))
			if !ok {
				return cli.Exit("--condition must be near_mint|lightly_played|heavily_played|damaged", 2)
			}
			req.Condition = cond
		}
		if cmd.IsSet("quantity-delta") {
			req.QuantityDelta = int32(cmd.Int("quantity-delta"))
		}
		setStringFlag(cmd, "item-id", &req.ItemID)
		setStringFlag(cmd, "client-mutation-id", &req.ClientMutationID)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.AdjustCollectionItemQuantityRequest) (any, error) {
		return c.AdjustCollectionItemQuantity(ctx, req)
	})
}

func newCollectionItemsTransferCommand() *cli.Command {
	return &cli.Command{
		Name:  "transfer",
		Usage: "Transfer an item to another collection",
		Flags: []cli.Flag{
			requestFileFlag(),
			&cli.StringFlag{Name: "id", Usage: "Item ID"},
			&cli.StringFlag{Name: "destination-collection-id", Usage: "Destination collection ID"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			payload, err := readObjectPayload(cmd)
			if err != nil {
				return err
			}
			setPayloadStringFlag(cmd, "destination-collection-id", "destinationCollectionId", payload)
			path := "/v1/collection_items/" + url.PathEscape(strings.TrimSpace(cmd.String("id"))) + ":transfer"
			return writeAuthenticatedJSON(ctx, cmd, http.MethodPost, path, payload)
		},
	}
}

func newCollectionItemsBatchGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "batch-get",
		Usage: "Batch get collection items",
		Flags: []cli.Flag{
			requestFileFlag(),
			repeatedIDsFlag("id", "Item ID (repeatable or comma-separated)"),
			&cli.BoolFlag{Name: "allow-partial", Usage: "Allow partial results"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			payload, err := readObjectPayload(cmd)
			if err != nil {
				return err
			}
			if cmd.IsSet("id") {
				payload["itemIds"] = splitCSV(cmd.StringSlice("id"))
			}
			setPayloadBoolFlag(cmd, "allow-partial", "allowPartial", payload)
			return writeAuthenticatedJSON(ctx, cmd, http.MethodPost, "/v1/collection_items:batchGet", payload)
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
