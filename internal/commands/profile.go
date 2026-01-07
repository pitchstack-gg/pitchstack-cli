package commands

import (
	"context"
	"strings"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"

	"github.com/urfave/cli/v3"
)

func newProfileCommand() *cli.Command {
	return &cli.Command{
		Name:  "profile",
		Usage: "User profile commands",
		Commands: []*cli.Command{
			newProfileGetCommand(),
			newProfileUpdateCommand(),
			newProfileSettingsCommand(),
			newProfileAvatarCommand(),
			newProfileSocialsCommand(),
		},
	}
}

func newProfileGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a user's profile",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "user-id", Usage: "User ID", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			resp, err := st.Service.GetProfile(ctx, &clientv1.GetProfileRequest{UserID: strings.TrimSpace(cmd.String("user-id"))})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newProfileUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update the authenticated user's profile",
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
		Usage: "Profile settings",
		Commands: []*cli.Command{
			newProfileSettingsGetCommand(),
			newProfileSettingsUpdateCommand(),
		},
	}
}

func newProfileSettingsGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get profile settings (current user)",
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
		Usage: "Update profile settings (current user)",
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
		Usage: "Avatar commands",
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
		},
	}
}

func newProfileSocialsCommand() *cli.Command {
	return &cli.Command{
		Name:  "socials",
		Usage: "Social profiles",
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
		Usage: "Get social profiles for a user",
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
		Usage: "Upsert a social profile for the current user",
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
		Usage: "Remove a social profile for the current user",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "platform", Usage: "Platform", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			resp, err := st.Service.RemoveSocialProfile(ctx, &clientv1.RemoveSocialProfileRequest{Platform: strings.TrimSpace(cmd.String("platform"))})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}
