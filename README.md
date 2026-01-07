# pitchstack-cli

CLI client for the Pitchstack API.

## Install

Prereqs: Go installed (`go version`).

From this repo:

```sh
# Installs into your Go bin dir
make install

# Or use the script (defaults: macOS -> /opt/homebrew/bin (if writable) else /usr/local/bin; Linux -> /usr/local/bin)
./scripts/install.sh

# If you need a custom location:
OUT_DIR="$HOME/bin" ./scripts/install.sh
```

### Install From GitHub Releases

Download the archive for your OS/arch from the GitHub Releases page and place `pitchstack` somewhere on your `PATH`:

```sh
tar -xzf pitchstack_<version>_<os>_<arch>.tar.gz
sudo install -m 0755 pitchstack /usr/local/bin/pitchstack
```

## Quickstart

1) Create a config (optional — defaults work without it):

```sh
go run ./cmd/pitchstack config init
```

2) Login:

```sh
go run ./cmd/pitchstack login --email you@example.com
```

3) Verify session:

```sh
go run ./cmd/pitchstack whoami
```

## Collections

```sh
go run ./cmd/pitchstack collections list
go run ./cmd/pitchstack collections create --name "My Binder" --type binder --visibility private
go run ./cmd/pitchstack collections get --id <collection-id>
go run ./cmd/pitchstack collections update --id <collection-id> --name "New Name"
go run ./cmd/pitchstack collections delete --id <collection-id>
```

## Collection Items

```sh
go run ./cmd/pitchstack collections items list --collection-id <collection-id>
go run ./cmd/pitchstack collections items add --collection-id <collection-id> --product-id <product-id> --quantity 1 --condition near_mint
go run ./cmd/pitchstack collections items get --id <item-id>
go run ./cmd/pitchstack collections items update --id <item-id> --quantity 2
go run ./cmd/pitchstack collections items delete --id <item-id>
```

## Sync

```sh
go run ./cmd/pitchstack sync changes --cursor <cursor> --page-size 100 --include-documents
go run ./cmd/pitchstack sync changes --cursor <cursor> --poll --interval 5s

# Apply local changes from a JSON file (either {deviceId, changes:[...]} or just an array of changes)
go run ./cmd/pitchstack sync apply --device-id device-1 --file ./changes.json

go run ./cmd/pitchstack sync subscriptions list
go run ./cmd/pitchstack sync subscriptions update --subscribe collection:<collection-id> --unsubscribe deck:<deck-id>
```

## Cards

```sh
go run ./cmd/pitchstack cards search --q "Fai"
go run ./cmd/pitchstack cards get --id <card-id>
go run ./cmd/pitchstack cards printings --card-id <card-id>
go run ./cmd/pitchstack cards printing --id <printing-id>
go run ./cmd/pitchstack cards product --id <product-id>
```

## Profile

```sh
go run ./cmd/pitchstack profile get --user-id <user-id>
go run ./cmd/pitchstack profile update --name "Your Name" --bio "Hello"
go run ./cmd/pitchstack profile settings get
go run ./cmd/pitchstack profile settings update --profile-visibility public --social-visibility followers
go run ./cmd/pitchstack profile avatar set --url https://example.com/avatar.png
go run ./cmd/pitchstack profile socials get --user-id <user-id>
go run ./cmd/pitchstack profile socials upsert --platform bluesky --handle you --url https://bsky.app/profile/you
go run ./cmd/pitchstack profile socials remove --platform bluesky
```

## Files

- Config: `$(user config dir)/pitchstack/config.json`
- Session (tokens): `$(user config dir)/pitchstack/session.json` (chmod 600)
