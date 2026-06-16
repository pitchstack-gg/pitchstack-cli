package commands

import (
	"io"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestCommandHelpDescriptionsUseUnifiedLanguage(t *testing.T) {
	t.Parallel()

	root := NewRootCommand(strings.NewReader(""), io.Discard, io.Discard)
	var walk func(path string, cmd *cli.Command)
	walk = func(path string, cmd *cli.Command) {
		t.Helper()
		usage := strings.ToLower(strings.TrimSpace(cmd.Usage))
		for _, banned := range []string{"helper", "helpers", " commands"} {
			if strings.Contains(usage, banned) {
				t.Fatalf("%s uses non-standard help text %q", path, cmd.Usage)
			}
		}
		for _, child := range cmd.Commands {
			walk(path+" "+child.Name, child)
		}
	}
	walk(root.Name, root)
}
