package commands

import (
	"context"
	"strings"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
	"github.com/urfave/cli/v3"
)

func newMeCommand() *cli.Command {
	return &cli.Command{
		Name:  "me",
		Usage: "Manage your Pitchstack account",
		Commands: []*cli.Command{
			newMeProfileCommand(),
			newMeSocialsCommand(),
			newMeNotificationsCommand(),
			newMePriceWatchesCommand(),
		},
	}
}

func newMeProfileCommand() *cli.Command {
	return &cli.Command{
		Name:  "profile",
		Usage: "Manage your profile",
		Commands: []*cli.Command{
			newMeProfileGetCommand(),
			newProfileUpdateCommand(),
			newProfileSettingsCommand(),
			newProfileAvatarCommand(),
			newProfileBackgroundCommand(),
			newMeProfilePinsCommand(),
			newProfilePrivacyCommand(),
		},
	}
}

func newMeProfileGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Show your profile",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withSDKClient(ctx, cmd, true, func(c *clientv1.Client) (any, error) {
				return c.GetMyProfile(ctx)
			})
		},
	}
}

func newMeSocialsCommand() *cli.Command {
	return &cli.Command{
		Name:  "socials",
		Usage: "Manage your social profiles",
		Commands: []*cli.Command{
			newMeSocialsGetCommand(),
			newProfileSocialsUpsertCommand(),
			newProfileSocialsRemoveCommand(),
		},
	}
}

func newMeProfilePinsCommand() *cli.Command {
	cmd := newProfilePinsCommand()
	if len(cmd.Commands) > 0 {
		cmd.Commands[0] = newMeProfilePinsGetCommand()
	}
	return cmd
}

func newMeProfilePinsGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get your pinned resources",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			userID, err := currentSessionUserID(ctx, st)
			if err != nil {
				return err
			}
			c, err := st.Service.AuthenticatedClient()
			if err != nil {
				return err
			}
			resp, err := c.GetPinnedResources(ctx, &clientv1.GetPinnedResourcesRequest{UserID: strings.TrimSpace(userID)})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newMeSocialsGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "List your social profiles",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			userID, err := currentSessionUserID(ctx, st)
			if err != nil {
				return err
			}
			resp, err := st.Service.GetSocialProfiles(ctx, &clientv1.GetSocialProfilesRequest{UserID: strings.TrimSpace(userID)})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}
