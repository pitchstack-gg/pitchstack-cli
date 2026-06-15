# pitchstack-cli

Customer CLI for the Pitchstack API.

## Install

Install the latest macOS or Linux release:

```sh
curl -fsSL https://raw.githubusercontent.com/pitchstack-gg/pitchstack-cli/main/scripts/install-latest.sh | sh
```

Install somewhere else:

```sh
curl -fsSL https://raw.githubusercontent.com/pitchstack-gg/pitchstack-cli/main/scripts/install-latest.sh | INSTALL_DIR="$HOME/bin" sh
```

The installer downloads the matching release archive, verifies it against the release checksums, installs `pitchstack`, and prints the installed version.

Manual install:

```sh
tar -xzf pitchstack_<version>_<os>_<arch>.tar.gz
sudo install -m 0755 pitchstack /usr/local/bin/pitchstack
pitchstack version
```

Update by running the install command again. Uninstall by removing the installed binary, for example:

```sh
sudo rm -f /usr/local/bin/pitchstack
```

## Quickstart

Defaults work without a config file. To write one explicitly:

```sh
pitchstack config init
pitchstack config show
```

Log in with browser-based OAuth:

```sh
pitchstack login
pitchstack whoami
pitchstack auth status
```

Log out and clear the local session:

```sh
pitchstack logout
```

## Common Commands

Cards:

```sh
pitchstack cards search --q "Fai"
pitchstack cards get --id <card-id>
pitchstack cards printings --card-id <card-id>
pitchstack cards product --id <product-id>
```

Collections:

```sh
pitchstack collections list
pitchstack collections create --name "My Binder" --type binder --visibility private
pitchstack collections get --id <collection-id>
pitchstack collections update --id <collection-id> --name "New Name"
pitchstack collections items add --collection-id <collection-id> --product-id <product-id> --quantity 1 --condition near_mint
pitchstack collections delete --id <collection-id>
```

Destructive commands prompt before changing or deleting data. Use `--yes` only in scripts where the target ID is already known:

```sh
pitchstack collections delete --id <collection-id> --yes
```

Decks:

```sh
pitchstack decks list
pitchstack decks get --id <deck-id>
pitchstack decks create --name "My Deck" --hero-id <hero-id> --format blitz
pitchstack decks delete --id <deck-id>
```

Profile:

```sh
pitchstack profile get
pitchstack profile update --name "Your Name" --bio "Hello"
pitchstack profile socials upsert --platform bluesky --handle you --url https://bsky.app/profile/you
pitchstack profile socials remove --platform bluesky
```

Auth helpers:

```sh
pitchstack auth me
pitchstack auth api-keys list
pitchstack auth api-keys create --name "automation"
pitchstack auth password request-reset --email you@example.com
pitchstack auth password reset
```

Password and token prompts do not echo secrets. For automation, pass request JSON with `--file` or `--file -` instead of putting secrets in shell history.

## Files

- Config: `$(user config dir)/pitchstack/config.json`
- Sessions: `$(user config dir)/pitchstack/sessions/<profile>.json` with mode `0600`
- Card cache: `$(user cache dir)/pitchstack/<profile>/cards/`
- Sync cache: `$(user cache dir)/pitchstack/<profile>/sync/`

Each config profile supports:

- `baseUrl`: API base URL, default `https://api.pitchstack.gg`
- `oauthBaseUrl`: OAuth web base URL, default `https://auth.pitchstack.gg`
- `timeoutSeconds`: request timeout
- `cardsDbUrl`, `cardsDbLastUpdatedUrl`, `cardsDbRefreshInterval`: local card data settings
- `powerSyncUrl`, `syncEnabled`: local sync settings

## Development

From this repository:

```sh
make test
make build
```

Install a local development build:

```sh
make install
```

Validate release configuration:

```sh
make release-check
make release-snapshot
```

Publishing a release is handled by GitHub Actions when a version tag is pushed:

```sh
git tag v0.1.0
git push origin v0.1.0
```
