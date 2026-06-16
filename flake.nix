{
  description = "Pitchstack CLI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {
        inherit system;
        config.allowUnfree = true;
      };

      version = self.shortRev or self.dirtyShortRev or "dev";

      pitchstack = pkgs.buildGoModule {
        pname = "pitchstack";
        inherit version;

        src = ./.;
        subPackages = ["cmd/pitchstack"];
        vendorHash = "sha256-in0S3C+OoDTf5mPKnfJR0xTuj4pgjDtW881ZIEUJfzM=";

        env.CGO_ENABLED = "0";

        ldflags = [
          "-s"
          "-w"
          "-X github.com/pitchstack-gg/pitchstack-cli/internal/buildinfo.Version=${version}"
          "-X github.com/pitchstack-gg/pitchstack-cli/internal/buildinfo.Commit=${version}"
        ];

        meta = {
          description = "Customer CLI for the Pitchstack API";
          homepage = "https://github.com/pitchstack-gg/pitchstack-cli";
          license = pkgs.lib.licenses.unfree;
          mainProgram = "pitchstack";
        };
      };
    in {
      packages = {
        default = pitchstack;
        pitchstack = pitchstack;
      };

      apps = {
        default = {
          type = "app";
          program = "${pkgs.lib.getExe pitchstack}";
          meta.description = "Run the Pitchstack CLI";
        };
        pitchstack = {
          type = "app";
          program = "${pkgs.lib.getExe pitchstack}";
          meta.description = "Run the Pitchstack CLI";
        };
      };

      devShells.default = pkgs.mkShell {
        packages = with pkgs; [
          go
          gopls
          gotools
          goreleaser
        ];
      };
    })
    // {
      overlays.default = final: prev: {
        pitchstack = self.packages.${prev.stdenv.hostPlatform.system}.default;
      };
    };
}
