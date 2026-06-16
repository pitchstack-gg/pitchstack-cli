package commands

import (
	"context"
	"os"
	"strings"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"

	"github.com/urfave/cli/v3"
)

func newProfileCommand() *cli.Command {
	return &cli.Command{
		Name:  "profile",
		Usage: "Manage user profiles",
		Commands: []*cli.Command{
			newProfileGetCommand(),
			newProfileSearchCommand(),
			newProfileUpdateCommand(),
			newProfileSettingsCommand(),
			newProfileAvatarCommand(),
			newProfileBackgroundCommand(),
			newProfilePinsCommand(),
			newProfilePrivacyCommand(),
			newProfileSocialsCommand(),
		},
	}
}

func newProfileGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Show a user profile",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "user-id", Usage: "User ID (optional)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			userID := strings.TrimSpace(cmd.String("user-id"))
			return withSDKClient(ctx, cmd, true, func(c *clientv1.Client) (any, error) {
				if userID == "" {
					return c.GetMyProfile(ctx)
				}
				return c.GetProfile(ctx, &clientv1.GetProfileRequest{UserID: userID})
			})
		},
	}
}

func newProfileSearchCommand() *cli.Command {
	return newSDKCommand("search", "Search users", append(pageFlags(), &cli.StringFlag{Name: "q", Usage: "Username prefix"}), true, func(cmd *cli.Command, req *clientv1.SearchUsersRequest) error {
		setStringFlag(cmd, "q", &req.SearchTerm)
		setPageFlags(cmd, &req.PageSize, &req.NextToken)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.SearchUsersRequest) (any, error) {
		return c.SearchUsers(ctx, req)
	})
}

func newProfileUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update current profile",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "username", Usage: "Username"},
			&cli.StringFlag{Name: "name", Usage: "Name"},
			&cli.StringFlag{Name: "avatar-url", Usage: "Avatar URL"},
			&cli.StringFlag{Name: "bio", Usage: "Bio"},
			&cli.StringFlag{Name: "location", Usage: "Location"},
			&cli.StringFlag{Name: "pronouns", Usage: "Pronouns"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			profile := &clientv1.UserProfile{}
			var mask []string

			if cmd.IsSet("username") {
				profile.Username = strings.TrimSpace(cmd.String("username"))
				mask = append(mask, "username")
			}
			if cmd.IsSet("name") {
				profile.Name = strings.TrimSpace(cmd.String("name"))
				mask = append(mask, "name")
			}
			if cmd.IsSet("avatar-url") {
				profile.AvatarURL = strings.TrimSpace(cmd.String("avatar-url"))
				mask = append(mask, "avatar_url")
			}
			if cmd.IsSet("bio") {
				profile.Bio = strings.TrimSpace(cmd.String("bio"))
				mask = append(mask, "bio")
			}
			if cmd.IsSet("location") {
				profile.Location = strings.TrimSpace(cmd.String("location"))
				mask = append(mask, "location")
			}
			if cmd.IsSet("pronouns") {
				profile.Pronouns = strings.TrimSpace(cmd.String("pronouns"))
				mask = append(mask, "pronouns")
			}

			if len(mask) == 0 {
				return cli.Exit("no updates provided", 2)
			}

			resp, err := st.Service.UpdateProfile(ctx, &clientv1.UpdateProfileRequest{
				Profile:    profile,
				UpdateMask: strings.Join(mask, ","),
			})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newProfileSettingsCommand() *cli.Command {
	return &cli.Command{
		Name:  "settings",
		Usage: "Manage profile settings",
		Commands: []*cli.Command{
			newProfileSettingsGetCommand(),
			newProfileSettingsUpdateCommand(),
		},
	}
}

func newProfileSettingsGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Show profile settings",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "user-id", Usage: "Optional user ID override (admin use)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			resp, err := st.Service.GetProfileSettings(ctx, &clientv1.GetProfileSettingsRequest{UserID: strings.TrimSpace(cmd.String("user-id"))})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newProfileSettingsUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update profile settings",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "profile-visibility", Usage: "Profile visibility (private|followers|public)"},
			&cli.StringFlag{Name: "social-visibility", Usage: "Social profiles visibility (private|followers|public)"},
			&cli.StringFlag{Name: "allow-messages", Usage: "Allow messages (ANYONE|FOLLOWERS|NONE)"},
			&cli.StringFlag{Name: "allow-trade-offers", Usage: "Allow trade offers (ANYONE|FOLLOWERS|NONE)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			settings := &clientv1.ProfileSettings{}
			var mask []string

			if cmd.IsSet("profile-visibility") {
				v, ok := parseProfileVisibility(cmd.String("profile-visibility"))
				if !ok {
					return cli.Exit("--profile-visibility must be private|followers|public", 2)
				}
				settings.ProfileVisibility = v
				mask = append(mask, "profile_visibility")
			}
			if cmd.IsSet("social-visibility") {
				v, ok := parseProfileVisibility(cmd.String("social-visibility"))
				if !ok {
					return cli.Exit("--social-visibility must be private|followers|public", 2)
				}
				settings.SocialProfilesVisibility = v
				mask = append(mask, "social_profiles_visibility")
			}
			if cmd.IsSet("allow-messages") {
				settings.AllowMessages = strings.TrimSpace(cmd.String("allow-messages"))
				mask = append(mask, "allow_messages")
			}
			if cmd.IsSet("allow-trade-offers") {
				settings.AllowTradeOffers = strings.TrimSpace(cmd.String("allow-trade-offers"))
				mask = append(mask, "allow_trade_offers")
			}

			if len(mask) == 0 {
				return cli.Exit("no updates provided", 2)
			}

			resp, err := st.Service.UpdateProfileSettings(ctx, &clientv1.UpdateProfileSettingsRequest{
				Settings:   settings,
				UpdateMask: strings.Join(mask, ","),
			})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func parseProfileVisibility(v string) (clientv1.ProfileVisibilityLevel, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "private":
		return clientv1.ProfileVisibilityLevelPrivate, true
	case "followers":
		return clientv1.ProfileVisibilityLevelFollowers, true
	case "public":
		return clientv1.ProfileVisibilityLevelPublic, true
	case "", "unspecified":
		return clientv1.ProfileVisibilityLevelUnspecified, false
	default:
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(v)), "VISIBILITY_LEVEL_") {
			return clientv1.ProfileVisibilityLevel(strings.ToUpper(strings.TrimSpace(v))), true
		}
		return clientv1.ProfileVisibilityLevelUnspecified, false
	}
}

func newProfileAvatarCommand() *cli.Command {
	return &cli.Command{
		Name:  "avatar",
		Usage: "Manage profile avatar",
		Commands: []*cli.Command{
			{
				Name:  "set",
				Usage: "Set avatar URL",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "url", Usage: "Avatar URL", Required: true},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					st, err := getState(ctx)
					if err != nil {
						return err
					}
					resp, err := st.Service.SetAvatarURL(ctx, &clientv1.SetAvatarURLRequest{AvatarURL: strings.TrimSpace(cmd.String("url"))})
					if err != nil {
						return err
					}
					return writeJSON(cmd.Writer, resp)
				},
			},
			newSDKCommand("begin", "Begin avatar upload", []cli.Flag{
				&cli.StringFlag{Name: "content-type", Usage: "Content type"},
				&cli.IntFlag{Name: "content-length", Usage: "Content length"},
			}, true, func(cmd *cli.Command, req *clientv1.BeginAvatarUploadRequest) error {
				setStringFlag(cmd, "content-type", &req.ContentType)
				setInt64Flag(cmd, "content-length", &req.ContentLength)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.BeginAvatarUploadRequest) (any, error) {
				return c.BeginAvatarUpload(ctx, req)
			}),
			newSDKCommand("complete", "Complete avatar upload", []cli.Flag{&cli.StringFlag{Name: "upload-id", Usage: "Upload ID"}}, true, func(cmd *cli.Command, req *clientv1.CompleteAvatarUploadRequest) error {
				setStringFlag(cmd, "upload-id", &req.UploadID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.CompleteAvatarUploadRequest) (any, error) {
				return c.CompleteAvatarUpload(ctx, req)
			}),
			newProfileAvatarUploadCommand(),
		},
	}
}

func newProfileAvatarUploadCommand() *cli.Command {
	return &cli.Command{
		Name:  "upload",
		Usage: "Upload and apply an avatar file",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "file", Usage: "File to upload", Required: true},
			&cli.StringFlag{Name: "content-type", Usage: "Content type", Value: "application/octet-stream"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			c, err := st.Service.AuthenticatedClient()
			if err != nil {
				return err
			}
			filePath := cmd.String("file")
			info, err := os.Stat(filePath)
			if err != nil {
				return err
			}
			contentLength := info.Size()
			contentType := cmd.String("content-type")
			begin, err := c.BeginAvatarUpload(ctx, &clientv1.BeginAvatarUploadRequest{ContentType: contentType, ContentLength: &contentLength})
			if err != nil {
				return err
			}
			if err := uploadFileToSignedURL(ctx, st.Service.HTTPClient(), begin.UploadURL, begin.RequiredHeaders, filePath, contentType); err != nil {
				return err
			}
			resp, err := c.CompleteAvatarUpload(ctx, &clientv1.CompleteAvatarUploadRequest{UploadID: begin.UploadID})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newProfileBackgroundCommand() *cli.Command {
	return &cli.Command{
		Name:  "background",
		Usage: "Manage profile background",
		Commands: []*cli.Command{
			newSDKCommand("begin", "Begin profile background upload", []cli.Flag{
				&cli.StringFlag{Name: "content-type", Usage: "Content type"},
				&cli.IntFlag{Name: "content-length", Usage: "Content length"},
			}, true, func(cmd *cli.Command, req *clientv1.BeginProfileBackgroundUploadRequest) error {
				setStringFlag(cmd, "content-type", &req.ContentType)
				setInt64Flag(cmd, "content-length", &req.ContentLength)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.BeginProfileBackgroundUploadRequest) (any, error) {
				return c.BeginProfileBackgroundUpload(ctx, req)
			}),
			newSDKCommand("complete", "Complete profile background upload", []cli.Flag{&cli.StringFlag{Name: "upload-id", Usage: "Upload ID"}}, true, func(cmd *cli.Command, req *clientv1.CompleteProfileBackgroundUploadRequest) error {
				setStringFlag(cmd, "upload-id", &req.UploadID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.CompleteProfileBackgroundUploadRequest) (any, error) {
				return c.CompleteProfileBackgroundUpload(ctx, req)
			}),
			{
				Name:  "clear",
				Usage: "Clear profile background",
				Flags: []cli.Flag{yesFlag()},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if err := confirmAction(cmd, "Clear", "profile background", "current user"); err != nil {
						return err
					}
					return withSDKClient(ctx, cmd, true, func(c *clientv1.Client) (any, error) {
						return c.ClearProfileBackground(ctx)
					})
				},
			},
			newProfileBackgroundUploadCommand(),
		},
	}
}

func newProfileBackgroundUploadCommand() *cli.Command {
	return &cli.Command{
		Name:  "upload",
		Usage: "Upload and apply a profile background file",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "file", Usage: "File to upload", Required: true},
			&cli.StringFlag{Name: "content-type", Usage: "Content type", Value: "application/octet-stream"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			c, err := st.Service.AuthenticatedClient()
			if err != nil {
				return err
			}
			filePath := cmd.String("file")
			info, err := os.Stat(filePath)
			if err != nil {
				return err
			}
			contentLength := info.Size()
			contentType := cmd.String("content-type")
			begin, err := c.BeginProfileBackgroundUpload(ctx, &clientv1.BeginProfileBackgroundUploadRequest{ContentType: contentType, ContentLength: &contentLength})
			if err != nil {
				return err
			}
			if err := uploadFileToSignedURL(ctx, st.Service.HTTPClient(), begin.UploadURL, begin.RequiredHeaders, filePath, contentType); err != nil {
				return err
			}
			resp, err := c.CompleteProfileBackgroundUpload(ctx, &clientv1.CompleteProfileBackgroundUploadRequest{UploadID: begin.UploadID})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newProfilePinsCommand() *cli.Command {
	return &cli.Command{
		Name:  "pins",
		Usage: "Manage pinned resources",
		Commands: []*cli.Command{
			newSDKCommand("get", "Get pinned resources", []cli.Flag{&cli.StringFlag{Name: "user-id", Usage: "User ID"}}, true, func(cmd *cli.Command, req *clientv1.GetPinnedResourcesRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetPinnedResourcesRequest) (any, error) {
				return c.GetPinnedResources(ctx, req)
			}),
			newSDKCommand("pin-collection", "Pin a collection", []cli.Flag{&cli.StringFlag{Name: "collection-id", Usage: "Collection ID"}}, true, func(cmd *cli.Command, req *clientv1.PinCollectionRequest) error {
				setStringFlag(cmd, "collection-id", &req.CollectionID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.PinCollectionRequest) (any, error) {
				return c.PinCollection(ctx, req)
			}),
			newSDKCommand("unpin-collection", "Unpin a collection", []cli.Flag{&cli.StringFlag{Name: "collection-id", Usage: "Collection ID"}}, true, func(cmd *cli.Command, req *clientv1.UnpinCollectionRequest) error {
				setStringFlag(cmd, "collection-id", &req.CollectionID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UnpinCollectionRequest) (any, error) {
				return c.UnpinCollection(ctx, req)
			}),
			newSDKCommand("pin-deck", "Pin a deck", []cli.Flag{&cli.StringFlag{Name: "deck-id", Usage: "Deck ID"}}, true, func(cmd *cli.Command, req *clientv1.PinDeckRequest) error {
				setStringFlag(cmd, "deck-id", &req.DeckID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.PinDeckRequest) (any, error) {
				return c.PinDeck(ctx, req)
			}),
			newSDKCommand("unpin-deck", "Unpin a deck", []cli.Flag{&cli.StringFlag{Name: "deck-id", Usage: "Deck ID"}}, true, func(cmd *cli.Command, req *clientv1.UnpinDeckRequest) error {
				setStringFlag(cmd, "deck-id", &req.DeckID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UnpinDeckRequest) (any, error) {
				return c.UnpinDeck(ctx, req)
			}),
		},
	}
}

func newProfilePrivacyCommand() *cli.Command {
	return &cli.Command{
		Name:  "privacy",
		Usage: "Manage privacy consent",
		Commands: []*cli.Command{
			newSDKNoRequestCommand("get", "Get privacy consent", true, func(ctx context.Context, c *clientv1.Client) (any, error) {
				return c.GetPrivacyConsent(ctx)
			}),
			newSDKCommand("update", "Update privacy consent", []cli.Flag{
				&cli.BoolFlag{Name: "analytics-allowed", Usage: "Analytics allowed"},
				&cli.IntFlag{Name: "consent-version", Usage: "Consent version"},
				&cli.StringFlag{Name: "source", Usage: "Source"},
				&cli.StringFlag{Name: "platform", Usage: "Platform"},
				&cli.StringFlag{Name: "app-version", Usage: "App version"},
				&cli.StringFlag{Name: "device-id-hash", Usage: "Device ID hash"},
				&cli.StringFlag{Name: "client-action-at", Usage: "Client action time (RFC3339)"},
				&cli.StringFlag{Name: "ad-consent-provider", Usage: "Ad consent provider"},
				&cli.BoolFlag{Name: "ad-consent-region-applies", Usage: "Whether ad consent region applies"},
				&cli.StringFlag{Name: "ad-consent-last-seen-at", Usage: "Ad consent last seen time (RFC3339)"},
			}, true, func(cmd *cli.Command, req *clientv1.UpdatePrivacyConsentRequest) error {
				if cmd.IsSet("analytics-allowed") {
					req.AnalyticsAllowed = cmd.Bool("analytics-allowed")
				}
				if cmd.IsSet("consent-version") {
					req.ConsentVersion = int32(cmd.Int("consent-version"))
				}
				setStringFlag(cmd, "source", &req.Source)
				setStringFlag(cmd, "platform", &req.Platform)
				setStringFlag(cmd, "app-version", &req.AppVersion)
				setStringFlag(cmd, "device-id-hash", &req.DeviceIDHash)
				if err := setTimeFlag(cmd, "client-action-at", &req.ClientActionAt); err != nil {
					return err
				}
				setStringFlag(cmd, "ad-consent-provider", &req.AdConsentProvider)
				if cmd.IsSet("ad-consent-region-applies") {
					req.AdConsentRegionApplies = cmd.Bool("ad-consent-region-applies")
				}
				return setTimeFlag(cmd, "ad-consent-last-seen-at", &req.AdConsentLastSeenAt)
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UpdatePrivacyConsentRequest) (any, error) {
				return c.UpdatePrivacyConsent(ctx, req)
			}),
		},
	}
}

func newProfileSocialsCommand() *cli.Command {
	return &cli.Command{
		Name:  "socials",
		Usage: "Manage social profiles",
		Commands: []*cli.Command{
			newProfileSocialsGetCommand(),
			newProfileSocialsUpsertCommand(),
			newProfileSocialsRemoveCommand(),
		},
	}
}

func newProfileSocialsGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "List social profiles",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "user-id", Usage: "User ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			resp, err := st.Service.GetSocialProfiles(ctx, &clientv1.GetSocialProfilesRequest{UserID: strings.TrimSpace(cmd.String("user-id"))})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newProfileSocialsUpsertCommand() *cli.Command {
	return &cli.Command{
		Name:  "upsert",
		Usage: "Save a social profile",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "platform", Usage: "Platform", Required: true},
			&cli.StringFlag{Name: "handle", Usage: "Handle"},
			&cli.StringFlag{Name: "url", Usage: "URL"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			resp, err := st.Service.UpsertSocialProfile(ctx, &clientv1.UpsertSocialProfileRequest{
				Platform: strings.TrimSpace(cmd.String("platform")),
				Handle:   strings.TrimSpace(cmd.String("handle")),
				URL:      strings.TrimSpace(cmd.String("url")),
			})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newProfileSocialsRemoveCommand() *cli.Command {
	return &cli.Command{
		Name:  "remove",
		Usage: "Remove a social profile",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "platform", Usage: "Platform", Required: true},
			yesFlag(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			platform := strings.TrimSpace(cmd.String("platform"))
			if err := confirmAction(cmd, "Remove", "social profile", platform); err != nil {
				return err
			}
			resp, err := st.Service.RemoveSocialProfile(ctx, &clientv1.RemoveSocialProfileRequest{Platform: platform})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}
