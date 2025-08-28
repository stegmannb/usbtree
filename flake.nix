{
  description = "USBTree - A cross-platform CLI tool for displaying USB devices in a tree structure";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        # Main package definition
        usbtree = pkgs.buildGoModule rec {
          pname = "usbtree";
          version = "1.1.0";

          src = ./.;

          vendorHash = "sha256-R8PGiiPyDzHWXgIFeJXQ28dJd0jDr8FZM7HajodDv0Y=";

          # No longer need libusb
          nativeBuildInputs = [ ];
          buildInputs = [ ];

          # CGO no longer required
          env.CGO_ENABLED = "0";

          # Build flags with version injection
          ldflags = [
            "-s"
            "-w"
            "-X github.com/stegmannb/usbtree/cmd.version=${version}"
          ];

          meta = with pkgs.lib; {
            description = "A cross-platform CLI tool that displays connected USB devices in a hierarchical tree structure";
            homepage = "https://github.com/stegmannb/usbtree";
            license = licenses.mit;
            maintainers = [ ];
            mainProgram = "usbtree";
            platforms = platforms.unix;
          };

          # Run tests during build
          checkPhase = ''
            runHook preCheck
            go test ./...
            runHook postCheck
          '';

          doCheck = true;
        };

      in
      {
        # Development shell
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go development tools
            go
            gopls
            gotools
            go-tools
            delve

            # Build dependencies (no longer need libusb)

            # Additional dev tools
            git
          ];

          shellHook = ''
            echo "╔══════════════════════════════════════╗"
            echo "║   USBTree Development Environment    ║"
            echo "╚══════════════════════════════════════╝"
            echo ""
            echo "📦 Available commands:"
            echo "  go run .              - Run the application"
            echo "  go test ./...         - Run all tests"
            echo "  go build -o usbtree . - Build binary"
            echo "  nix build             - Build with Nix"
            echo "  nix run               - Run with Nix"
            echo ""
            echo "🔧 System info:"
            echo "  libusb:     Not required (using native OS commands)"
            echo "  Go version: $(go version | cut -d' ' -f3)"
            echo "  Platform:   ${system}"
            echo ""
          '';

          # CGO no longer needed
          CGO_ENABLED = "0";
        };

        # Packages
        packages = {
          default = usbtree;
          usbtree = usbtree;
        };

        # Apps - for easy running with nix run
        apps.default = flake-utils.lib.mkApp {
          drv = usbtree;
          name = "usbtree";
        };

        # Checks - run tests
        checks = {
          usbtree-tests = pkgs.runCommand "usbtree-tests"
            {
              nativeBuildInputs = with pkgs; [
                go
              ];
              buildInputs = [ ];
              src = ./.;
            } ''
            cd $src
            export CGO_ENABLED=0
            go test ./...
            touch $out
          '';
        };

        # Formatter
        formatter = pkgs.nixpkgs-fmt;
      });
}
