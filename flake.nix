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
        usbtree = pkgs.buildGoModule {
          pname = "usbtree";
          version = "1.0.0";

          src = ./.;

          vendorHash = "sha256-AFdBY34NEZIY781dmA76h7cMfRdsv5xu2RyeYGH+ZLw=";

          nativeBuildInputs = [ pkgs.pkg-config ];
          buildInputs = [ pkgs.libusb1 ];

          # Enable CGO for libusb
          env.CGO_ENABLED = "1";

          # Build flags
          ldflags = [
            "-s"
            "-w"
          ];

          meta = with pkgs.lib; {
            description = "A cross-platform CLI tool that displays connected USB devices in a hierarchical tree structure";
            homepage = "https://github.com/user/usbtree";
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
            
            # Build dependencies
            libusb1
            pkg-config
            
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
            echo "  libusb-1.0: $(pkg-config --modversion libusb-1.0 2>/dev/null || echo 'not found')"
            echo "  Go version: $(go version | cut -d' ' -f3)"
            echo "  Platform:   ${system}"
            echo ""
          '';

          # Set environment variables for CGO
          CGO_ENABLED = "1";
          PKG_CONFIG_PATH = "${pkgs.libusb1.dev}/lib/pkgconfig";
          LD_LIBRARY_PATH = "${pkgs.libusb1.out}/lib";
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
          usbtree-tests = pkgs.runCommand "usbtree-tests" {
            nativeBuildInputs = with pkgs; [
              go
              pkg-config
            ];
            buildInputs = [ pkgs.libusb1 ];
            src = ./.;
          } ''
            cd $src
            export CGO_ENABLED=1
            export PKG_CONFIG_PATH="${pkgs.libusb1.dev}/lib/pkgconfig"
            go test ./...
            touch $out
          '';
        };

        # Formatter
        formatter = pkgs.nixpkgs-fmt;
      });
}