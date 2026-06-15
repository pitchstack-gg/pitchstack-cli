package commands

import (
	"context"
	"encoding/base64"
	"net/http"
	"os"
	"strings"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
	"github.com/urfave/cli/v3"
)

func newAuthMeCommand() *cli.Command {
	return &cli.Command{
		Name:  "me",
		Usage: "Show raw current user and access profile",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return writeAuthenticatedJSON(ctx, cmd, http.MethodGet, "/v1/me", nil)
		},
	}
}

func newAuthAPIKeysCommand() *cli.Command {
	return &cli.Command{
		Name:  "api-keys",
		Usage: "Manage API keys",
		Commands: []*cli.Command{
			newSDKCommand("list", "List API keys", pageFlags(), true, func(cmd *cli.Command, req *clientv1.ListAPIKeysRequest) error {
				setPageFlags(cmd, &req.PageSize, &req.NextToken)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListAPIKeysRequest) (any, error) {
				return c.ListAPIKeys(ctx, req)
			}),
			newSDKCommand("create", "Create an API key", []cli.Flag{
				&cli.StringFlag{Name: "name", Usage: "API key name"},
				repeatedIDsFlag("scope", "Scope (repeatable or comma-separated)"),
				&cli.IntFlag{Name: "rate-limit-per-minute", Usage: "Rate limit per minute"},
				&cli.StringFlag{Name: "expires-at", Usage: "Expiration time (RFC3339)"},
			}, true, func(cmd *cli.Command, req *clientv1.CreateAPIKeyRequest) error {
				setStringFlag(cmd, "name", &req.Name)
				if cmd.IsSet("scope") {
					req.Scopes = splitCSV(cmd.StringSlice("scope"))
				}
				setInt32Flag(cmd, "rate-limit-per-minute", &req.RateLimitPerMinute)
				return setTimeFlag(cmd, "expires-at", &req.ExpiresAt)
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.CreateAPIKeyRequest) (any, error) {
				return c.CreateAPIKey(ctx, req)
			}),
			newSDKCommand("validate", "Validate an API key", []cli.Flag{
				&cli.StringFlag{Name: "api-key", Usage: "Plaintext API key"},
			}, false, func(cmd *cli.Command, req *clientv1.ValidateAPIKeyRequest) error {
				setStringFlag(cmd, "api-key", &req.APIKey)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ValidateAPIKeyRequest) (any, error) {
				return c.ValidateAPIKey(ctx, req)
			}),
			newSDKCommand("revoke", "Revoke an API key", []cli.Flag{
				&cli.StringFlag{Name: "id", Usage: "API key ID"},
			}, true, func(cmd *cli.Command, req *clientv1.RevokeAPIKeyRequest) error {
				setStringFlag(cmd, "id", &req.APIKeyID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.RevokeAPIKeyRequest) (any, error) {
				return c.RevokeAPIKey(ctx, req)
			}),
		},
	}
}

func newAuthPasswordCommand() *cli.Command {
	return &cli.Command{
		Name:  "password",
		Usage: "Password helpers",
		Commands: []*cli.Command{
			newSDKCommand("change", "Change a user's password", []cli.Flag{
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
				&cli.StringFlag{Name: "current-password", Usage: "Current password"},
				&cli.StringFlag{Name: "new-password", Usage: "New password"},
			}, true, func(cmd *cli.Command, req *clientv1.ChangePasswordRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				setStringFlag(cmd, "current-password", &req.CurrentPassword)
				setStringFlag(cmd, "new-password", &req.NewPassword)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ChangePasswordRequest) (any, error) {
				return c.ChangePassword(ctx, req)
			}),
			newSDKCommand("request-reset", "Request a password reset", []cli.Flag{
				&cli.StringFlag{Name: "email", Usage: "Account email"},
			}, false, func(cmd *cli.Command, req *clientv1.RequestPasswordResetRequest) error {
				setStringFlag(cmd, "email", &req.Email)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.RequestPasswordResetRequest) (any, error) {
				return c.RequestPasswordReset(ctx, req)
			}),
			newSDKCommand("reset", "Complete a password reset", []cli.Flag{
				&cli.StringFlag{Name: "reset-token", Usage: "Reset token"},
				&cli.StringFlag{Name: "new-password", Usage: "New password"},
			}, false, func(cmd *cli.Command, req *clientv1.ResetPasswordRequest) error {
				setStringFlag(cmd, "reset-token", &req.ResetToken)
				setStringFlag(cmd, "new-password", &req.NewPassword)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ResetPasswordRequest) (any, error) {
				return c.ResetPassword(ctx, req)
			}),
			newSDKCommand("resend-verification", "Resend a verification email", []cli.Flag{
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
			}, true, func(cmd *cli.Command, req *clientv1.ResendVerificationEmailRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ResendVerificationEmailRequest) (any, error) {
				return c.ResendVerificationEmail(ctx, req)
			}),
			newSDKCommand("verify-email", "Verify an email address", []cli.Flag{
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
				&cli.StringFlag{Name: "verification-token", Usage: "Verification token"},
			}, false, func(cmd *cli.Command, req *clientv1.VerifyEmailRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				setStringFlag(cmd, "verification-token", &req.VerificationToken)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.VerifyEmailRequest) (any, error) {
				return c.VerifyEmail(ctx, req)
			}),
		},
	}
}

func newAuthMethodsCommand() *cli.Command {
	return &cli.Command{
		Name:  "methods",
		Usage: "Manage auth methods",
		Commands: []*cli.Command{
			newSDKCommand("list", "List auth methods", []cli.Flag{&cli.StringFlag{Name: "user-id", Usage: "User ID"}}, true, func(cmd *cli.Command, req *clientv1.ListAuthMethodsRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListAuthMethodsRequest) (any, error) {
				return c.ListAuthMethods(ctx, req)
			}),
			newSDKCommand("remove", "Remove an auth method", []cli.Flag{&cli.StringFlag{Name: "user-id", Usage: "User ID"}, &cli.StringFlag{Name: "method-type", Usage: "Auth method type"}}, true, func(cmd *cli.Command, req *clientv1.RemoveAuthMethodRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				if cmd.IsSet("method-type") {
					req.MethodType = parseAuthMethodType(cmd.String("method-type"))
				}
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.RemoveAuthMethodRequest) (any, error) {
				return c.RemoveAuthMethod(ctx, req)
			}),
			newSDKCommand("preferred", "Set preferred auth method", []cli.Flag{&cli.StringFlag{Name: "user-id", Usage: "User ID"}, &cli.StringFlag{Name: "method-type", Usage: "Auth method type"}}, true, func(cmd *cli.Command, req *clientv1.SetPreferredAuthMethodRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				if cmd.IsSet("method-type") {
					req.MethodType = parseAuthMethodType(cmd.String("method-type"))
				}
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.SetPreferredAuthMethodRequest) (any, error) {
				return c.SetPreferredAuthMethod(ctx, req)
			}),
		},
	}
}

func newAuthOAuthCommand() *cli.Command {
	return &cli.Command{
		Name:  "oauth",
		Usage: "OAuth provider helpers",
		Commands: []*cli.Command{
			newSDKCommand("initiate", "Initiate OAuth", []cli.Flag{
				&cli.StringFlag{Name: "provider", Usage: "Provider"},
				&cli.StringFlag{Name: "redirect-uri", Usage: "Redirect URI"},
				&cli.StringFlag{Name: "prompt", Usage: "OAuth prompt"},
			}, false, func(cmd *cli.Command, req *clientv1.InitiateOAuthRequest) error {
				setStringFlag(cmd, "provider", &req.Provider)
				setStringFlag(cmd, "redirect-uri", &req.RedirectURI)
				setStringFlag(cmd, "prompt", &req.Prompt)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.InitiateOAuthRequest) (any, error) {
				return c.InitiateOAuth(ctx, req)
			}),
			newSDKCommand("complete", "Complete OAuth", []cli.Flag{
				&cli.StringFlag{Name: "provider", Usage: "Provider"},
				&cli.StringFlag{Name: "code", Usage: "OAuth code"},
				&cli.StringFlag{Name: "state", Usage: "OAuth state"},
				&cli.StringFlag{Name: "redirect-uri", Usage: "Redirect URI"},
			}, false, func(cmd *cli.Command, req *clientv1.CompleteOAuthRequest) error {
				setStringFlag(cmd, "provider", &req.Provider)
				setStringFlag(cmd, "code", &req.Code)
				setStringFlag(cmd, "state", &req.State)
				setStringFlag(cmd, "redirect-uri", &req.RedirectURI)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.CompleteOAuthRequest) (any, error) {
				return c.CompleteOAuth(ctx, req)
			}),
			newSDKCommand("link", "Link an OAuth provider", []cli.Flag{
				&cli.StringFlag{Name: "provider", Usage: "Provider"},
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
				&cli.StringFlag{Name: "code", Usage: "OAuth code"},
				&cli.StringFlag{Name: "state", Usage: "OAuth state"},
				&cli.StringFlag{Name: "redirect-uri", Usage: "Redirect URI"},
			}, true, func(cmd *cli.Command, req *clientv1.LinkOAuthProviderRequest) error {
				setStringFlag(cmd, "provider", &req.Provider)
				setStringFlag(cmd, "user-id", &req.UserID)
				setStringFlag(cmd, "code", &req.Code)
				setStringFlag(cmd, "state", &req.State)
				setStringFlag(cmd, "redirect-uri", &req.RedirectURI)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.LinkOAuthProviderRequest) (any, error) {
				return c.LinkOAuthProvider(ctx, req)
			}),
			newSDKCommand("unlink", "Unlink an OAuth provider", []cli.Flag{
				&cli.StringFlag{Name: "provider", Usage: "Provider"},
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
			}, true, func(cmd *cli.Command, req *clientv1.UnlinkOAuthProviderRequest) error {
				setStringFlag(cmd, "provider", &req.Provider)
				setStringFlag(cmd, "user-id", &req.UserID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UnlinkOAuthProviderRequest) (any, error) {
				return c.UnlinkOAuthProvider(ctx, req)
			}),
		},
	}
}

func newAuthPatreonCommand() *cli.Command {
	return &cli.Command{
		Name:  "patreon",
		Usage: "Patreon account linking",
		Commands: []*cli.Command{
			{
				Name:  "initiate",
				Usage: "Initiate Patreon linking",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return writeAuthenticatedJSON(ctx, cmd, http.MethodPost, "/v1/auth/patreon/link:initiate", map[string]any{})
				},
			},
			{
				Name:  "complete",
				Usage: "Complete Patreon linking",
				Flags: []cli.Flag{
					requestFileFlag(),
					&cli.StringFlag{Name: "code", Usage: "OAuth code"},
					&cli.StringFlag{Name: "state", Usage: "OAuth state"},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					payload, err := readObjectPayload(cmd)
					if err != nil {
						return err
					}
					setPayloadStringFlag(cmd, "code", "code", payload)
					setPayloadStringFlag(cmd, "state", "state", payload)
					return writeAuthenticatedJSON(ctx, cmd, http.MethodPost, "/v1/auth/patreon/link:complete", payload)
				},
			},
			{
				Name:  "unlink",
				Usage: "Unlink Patreon",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if err := callAuthenticatedJSON(ctx, cmd, http.MethodDelete, "/v1/auth/patreon/link", nil, nil); err != nil {
						return err
					}
					return writeJSON(cmd.Writer, map[string]any{"unlinked": true})
				},
			},
		},
	}
}

func newAuthInternalCommand() *cli.Command {
	return &cli.Command{
		Name:  "internal",
		Usage: "Internal/admin auth helpers",
		Commands: []*cli.Command{
			newAuthInternalAccessProfileCommand(),
			newAuthInternalAddRoleCommand(),
			newAuthInternalRemoveRoleCommand(),
			newAuthInternalGrantEntitlementCommand(),
			newAuthInternalRevokeEntitlementCommand(),
			newAuthInternalSetLimitCommand(),
			newAuthInternalClearLimitCommand(),
			newAuthInternalProcessPatreonWebhookCommand(),
			newAuthInternalReconcilePatreonCommand(),
		},
	}
}

func newAuthInternalAccessProfileCommand() *cli.Command {
	return &cli.Command{
		Name:  "access-profile",
		Usage: "Get a user's access profile",
		Flags: []cli.Flag{
			requestFileFlag(),
			&cli.StringFlag{Name: "user-id", Usage: "User ID"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			payload, err := readObjectPayload(cmd)
			if err != nil {
				return err
			}
			setPayloadStringFlag(cmd, "user-id", "userId", payload)
			return writeAuthenticatedJSON(ctx, cmd, http.MethodPost, "/auth.v1.AuthInternalService/GetAccessProfile", payload)
		},
	}
}

func newAuthInternalAddRoleCommand() *cli.Command {
	return &cli.Command{
		Name:  "add-role",
		Usage: "Add a user role",
		Flags: authInternalRoleFlags(),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			payload, err := readObjectPayload(cmd)
			if err != nil {
				return err
			}
			setAuthInternalRolePayload(cmd, payload)
			if err := callAuthenticatedJSON(ctx, cmd, http.MethodPost, "/auth.v1.AuthInternalService/AddUserRole", payload, nil); err != nil {
				return err
			}
			return writeJSON(cmd.Writer, map[string]any{"added": true})
		},
	}
}

func newAuthInternalRemoveRoleCommand() *cli.Command {
	return &cli.Command{
		Name:  "remove-role",
		Usage: "Remove a user role",
		Flags: authInternalRoleFlags(),
		Action: func(ctx context.Context, cmd *cli.Command) error {
			payload, err := readObjectPayload(cmd)
			if err != nil {
				return err
			}
			setAuthInternalRolePayload(cmd, payload)
			if err := callAuthenticatedJSON(ctx, cmd, http.MethodPost, "/auth.v1.AuthInternalService/RemoveUserRole", payload, nil); err != nil {
				return err
			}
			return writeJSON(cmd.Writer, map[string]any{"removed": true})
		},
	}
}

func authInternalRoleFlags() []cli.Flag {
	return []cli.Flag{
		requestFileFlag(),
		&cli.StringFlag{Name: "user-id", Usage: "User ID"},
		&cli.StringFlag{Name: "role", Usage: "Role"},
		&cli.StringFlag{Name: "assigned-by", Usage: "Assigner user ID"},
	}
}

func setAuthInternalRolePayload(cmd *cli.Command, payload map[string]any) {
	setPayloadStringFlag(cmd, "user-id", "userId", payload)
	setPayloadStringFlag(cmd, "role", "role", payload)
	setPayloadStringFlag(cmd, "assigned-by", "assignedBy", payload)
}

func newAuthInternalGrantEntitlementCommand() *cli.Command {
	return &cli.Command{
		Name:  "grant-entitlement",
		Usage: "Grant a user entitlement",
		Flags: []cli.Flag{
			requestFileFlag(),
			&cli.StringFlag{Name: "user-id", Usage: "User ID"},
			&cli.StringFlag{Name: "entitlement", Usage: "Entitlement key"},
			&cli.StringFlag{Name: "source", Usage: "Source"},
			&cli.StringFlag{Name: "expires-at", Usage: "Expiration time (RFC3339)"},
			&cli.StringFlag{Name: "assigned-by", Usage: "Assigner user ID"},
			repeatedIDsFlag("metadata", "Metadata key=value (repeatable or comma-separated)"),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			payload, err := readObjectPayload(cmd)
			if err != nil {
				return err
			}
			setPayloadStringFlag(cmd, "user-id", "userId", payload)
			setPayloadStringFlag(cmd, "entitlement", "entitlement", payload)
			setPayloadStringFlag(cmd, "source", "source", payload)
			setPayloadStringFlag(cmd, "expires-at", "expiresAt", payload)
			setPayloadStringFlag(cmd, "assigned-by", "assignedBy", payload)
			if cmd.IsSet("metadata") {
				metadata, err := parseStringMap(cmd.StringSlice("metadata"))
				if err != nil {
					return cli.Exit("--metadata must be key=value", 2)
				}
				payload["metadata"] = metadata
			}
			if err := callAuthenticatedJSON(ctx, cmd, http.MethodPost, "/auth.v1.AuthInternalService/GrantEntitlement", payload, nil); err != nil {
				return err
			}
			return writeJSON(cmd.Writer, map[string]any{"granted": true})
		},
	}
}

func newAuthInternalRevokeEntitlementCommand() *cli.Command {
	return &cli.Command{
		Name:  "revoke-entitlement",
		Usage: "Revoke a user entitlement",
		Flags: []cli.Flag{
			requestFileFlag(),
			&cli.StringFlag{Name: "user-id", Usage: "User ID"},
			&cli.StringFlag{Name: "entitlement", Usage: "Entitlement key"},
			&cli.StringFlag{Name: "assigned-by", Usage: "Assigner user ID"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			payload, err := readObjectPayload(cmd)
			if err != nil {
				return err
			}
			setPayloadStringFlag(cmd, "user-id", "userId", payload)
			setPayloadStringFlag(cmd, "entitlement", "entitlement", payload)
			setPayloadStringFlag(cmd, "assigned-by", "assignedBy", payload)
			if err := callAuthenticatedJSON(ctx, cmd, http.MethodPost, "/auth.v1.AuthInternalService/RevokeEntitlement", payload, nil); err != nil {
				return err
			}
			return writeJSON(cmd.Writer, map[string]any{"revoked": true})
		},
	}
}

func newAuthInternalSetLimitCommand() *cli.Command {
	return &cli.Command{
		Name:  "set-limit",
		Usage: "Set a user limit override",
		Flags: []cli.Flag{
			requestFileFlag(),
			&cli.StringFlag{Name: "user-id", Usage: "User ID"},
			&cli.StringFlag{Name: "limit-key", Usage: "Limit key"},
			&cli.IntFlag{Name: "value", Usage: "Limit value"},
			&cli.StringFlag{Name: "source", Usage: "Source"},
			&cli.StringFlag{Name: "expires-at", Usage: "Expiration time (RFC3339)"},
			&cli.StringFlag{Name: "assigned-by", Usage: "Assigner user ID"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			payload, err := readObjectPayload(cmd)
			if err != nil {
				return err
			}
			setPayloadStringFlag(cmd, "user-id", "userId", payload)
			setPayloadStringFlag(cmd, "limit-key", "limitKey", payload)
			setPayloadIntFlag(cmd, "value", "value", payload)
			setPayloadStringFlag(cmd, "source", "source", payload)
			setPayloadStringFlag(cmd, "expires-at", "expiresAt", payload)
			setPayloadStringFlag(cmd, "assigned-by", "assignedBy", payload)
			if err := callAuthenticatedJSON(ctx, cmd, http.MethodPost, "/auth.v1.AuthInternalService/SetUserLimitOverride", payload, nil); err != nil {
				return err
			}
			return writeJSON(cmd.Writer, map[string]any{"set": true})
		},
	}
}

func newAuthInternalClearLimitCommand() *cli.Command {
	return &cli.Command{
		Name:  "clear-limit",
		Usage: "Clear a user limit override",
		Flags: []cli.Flag{
			requestFileFlag(),
			&cli.StringFlag{Name: "user-id", Usage: "User ID"},
			&cli.StringFlag{Name: "limit-key", Usage: "Limit key"},
			&cli.StringFlag{Name: "assigned-by", Usage: "Assigner user ID"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			payload, err := readObjectPayload(cmd)
			if err != nil {
				return err
			}
			setPayloadStringFlag(cmd, "user-id", "userId", payload)
			setPayloadStringFlag(cmd, "limit-key", "limitKey", payload)
			setPayloadStringFlag(cmd, "assigned-by", "assignedBy", payload)
			if err := callAuthenticatedJSON(ctx, cmd, http.MethodPost, "/auth.v1.AuthInternalService/ClearUserLimitOverride", payload, nil); err != nil {
				return err
			}
			return writeJSON(cmd.Writer, map[string]any{"cleared": true})
		},
	}
}

func newAuthInternalProcessPatreonWebhookCommand() *cli.Command {
	return &cli.Command{
		Name:  "process-patreon-webhook",
		Usage: "Process a Patreon webhook payload",
		Flags: []cli.Flag{
			requestFileFlag(),
			&cli.StringFlag{Name: "raw-body", Usage: "Raw webhook body"},
			&cli.StringFlag{Name: "raw-body-file", Usage: "File containing raw webhook body"},
			&cli.StringFlag{Name: "signature", Usage: "Patreon signature"},
			&cli.StringFlag{Name: "trigger", Usage: "Patreon trigger"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			payload, err := readObjectPayload(cmd)
			if err != nil {
				return err
			}
			if cmd.IsSet("raw-body") {
				payload["rawBody"] = base64.StdEncoding.EncodeToString([]byte(cmd.String("raw-body")))
			}
			if cmd.IsSet("raw-body-file") {
				data, err := os.ReadFile(strings.TrimSpace(cmd.String("raw-body-file")))
				if err != nil {
					return err
				}
				payload["rawBody"] = base64.StdEncoding.EncodeToString(data)
			}
			setPayloadStringFlag(cmd, "signature", "signature", payload)
			setPayloadStringFlag(cmd, "trigger", "trigger", payload)
			return writeAuthenticatedJSON(ctx, cmd, http.MethodPost, "/auth.v1.AuthInternalService/ProcessPatreonWebhook", payload)
		},
	}
}

func newAuthInternalReconcilePatreonCommand() *cli.Command {
	return &cli.Command{
		Name:  "reconcile-patreon",
		Usage: "Reconcile Patreon entitlements",
		Flags: []cli.Flag{requestFileFlag()},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			payload, err := readObjectPayload(cmd)
			if err != nil {
				return err
			}
			return writeAuthenticatedJSON(ctx, cmd, http.MethodPost, "/auth.v1.AuthInternalService/ReconcilePatreonEntitlements", payload)
		},
	}
}

func newAuthTokensCommand() *cli.Command {
	return &cli.Command{
		Name:  "tokens",
		Usage: "Token helpers",
		Commands: []*cli.Command{
			newSDKCommand("validate", "Validate an access token", []cli.Flag{
				&cli.StringFlag{Name: "access-token", Usage: "Access token"},
			}, false, func(cmd *cli.Command, req *clientv1.ValidateTokenRequest) error {
				setStringFlag(cmd, "access-token", &req.AccessToken)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ValidateTokenRequest) (any, error) {
				return c.ValidateToken(ctx, req)
			}),
		},
	}
}

func newAuthUsersCommand() *cli.Command {
	return &cli.Command{
		Name:  "users",
		Usage: "User account helpers",
		Commands: []*cli.Command{
			newSDKCommand("get", "Get a user", []cli.Flag{&cli.StringFlag{Name: "id", Usage: "User ID"}}, true, func(cmd *cli.Command, req *clientv1.GetUserRequest) error {
				setStringFlag(cmd, "id", &req.UserID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetUserRequest) (any, error) {
				return c.GetUser(ctx, req)
			}),
			newSDKCommand("update", "Update a user", []cli.Flag{
				&cli.StringFlag{Name: "id", Usage: "User ID"},
				&cli.StringFlag{Name: "email", Usage: "Email"},
				repeatedIDsFlag("role", "Role (repeatable or comma-separated)"),
			}, true, func(cmd *cli.Command, req *clientv1.UpdateUserRequest) error {
				setStringFlag(cmd, "id", &req.UserID)
				setStringPtrFlag(cmd, "email", &req.Email)
				if cmd.IsSet("role") {
					req.Roles = splitCSV(cmd.StringSlice("role"))
				}
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UpdateUserRequest) (any, error) {
				return c.UpdateUser(ctx, req)
			}),
			newSDKCommand("delete", "Delete a user", []cli.Flag{&cli.StringFlag{Name: "id", Usage: "User ID"}}, true, func(cmd *cli.Command, req *clientv1.DeleteUserRequest) error {
				setStringFlag(cmd, "id", &req.UserID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.DeleteUserRequest) (any, error) {
				return c.DeleteUser(ctx, req)
			}),
		},
	}
}

func newAuthPasskeysCommand() *cli.Command {
	return &cli.Command{
		Name:  "passkeys",
		Usage: "Passkey helpers",
		Commands: []*cli.Command{
			newSDKCommand("initiate-registration", "Initiate passkey registration", []cli.Flag{
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
				&cli.StringFlag{Name: "display-name", Usage: "Display name"},
			}, true, func(cmd *cli.Command, req *clientv1.InitiatePasskeyRegistrationRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				setStringFlag(cmd, "display-name", &req.DisplayName)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.InitiatePasskeyRegistrationRequest) (any, error) {
				return c.InitiatePasskeyRegistration(ctx, req)
			}),
			newSDKCommand("complete-registration", "Complete passkey registration from raw WebAuthn JSON fields", nil, true, nil, func(ctx context.Context, c *clientv1.Client, req *clientv1.CompletePasskeyRegistrationRequest) (any, error) {
				return c.CompletePasskeyRegistration(ctx, req)
			}),
			newSDKCommand("initiate-authentication", "Initiate passkey authentication", []cli.Flag{&cli.StringFlag{Name: "email", Usage: "Account email"}}, false, func(cmd *cli.Command, req *clientv1.InitiatePasskeyAuthenticationRequest) error {
				setStringFlag(cmd, "email", &req.Email)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.InitiatePasskeyAuthenticationRequest) (any, error) {
				return c.InitiatePasskeyAuthentication(ctx, req)
			}),
			newSDKCommand("complete-authentication", "Complete passkey authentication from raw WebAuthn JSON fields", nil, false, nil, func(ctx context.Context, c *clientv1.Client, req *clientv1.CompletePasskeyAuthenticationRequest) (any, error) {
				return c.CompletePasskeyAuthentication(ctx, req)
			}),
			newSDKCommand("list", "List user passkeys", []cli.Flag{&cli.StringFlag{Name: "user-id", Usage: "User ID"}}, true, func(cmd *cli.Command, req *clientv1.ListUserPasskeysRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListUserPasskeysRequest) (any, error) {
				return c.ListUserPasskeys(ctx, req)
			}),
			newSDKCommand("delete", "Delete a passkey", []cli.Flag{
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
				&cli.StringFlag{Name: "credential-id", Usage: "Credential ID"},
			}, true, func(cmd *cli.Command, req *clientv1.DeletePasskeyRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				setStringFlag(cmd, "credential-id", &req.CredentialID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.DeletePasskeyRequest) (any, error) {
				return c.DeletePasskey(ctx, req)
			}),
		},
	}
}

func parseAuthMethodType(v string) clientv1.AuthMethodType {
	switch v {
	case "password":
		return clientv1.AuthMethodTypePassword
	case "google":
		return clientv1.AuthMethodTypeOAuthGoogle
	case "discord":
		return clientv1.AuthMethodTypeOAuthDiscord
	case "apple":
		return clientv1.AuthMethodTypeOAuthApple
	case "passkey":
		return clientv1.AuthMethodTypePasskey
	default:
		return clientv1.AuthMethodType(v)
	}
}
