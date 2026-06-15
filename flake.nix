{
  description = "ashpipe — cd into a directory, get an SSH session";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachSystem [
      "x86_64-linux"
      "aarch64-linux"
      "aarch64-darwin"
      "x86_64-darwin"
    ] (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        isMacOS = pkgs.stdenv.isDarwin;
      in {
        packages.default = pkgs.buildGoModule {
          pname = "ashpipe";
          version = "0.1.0";
          src = ./.;
          vendorHash = null;  # vendor/ directory is included in source
          ldflags = [ "-s" "-w" "-X main.version=0.1.0" ];
          meta = {
            description = "cd into a directory, get an SSH session — for humans and AI agents";
            homepage = "https://github.com/KirisameLonnet/ashpipe";
            license = pkgs.lib.licenses.mit;
            mainProgram = "ashpipe";
          };
        };

        # Usage in home-manager after adding this flake as input:
        #
        #   inputs.ashpipe.url = "github:KirisameLonnet/ashpipe";
        #
        #   home.packages = [ inputs.ashpipe.packages.${system}.default ];
        #   programs.zsh.initExtra = ''
        #     eval "$(ashpipe hook zsh)"
        #   '';
        #
        # Because ashpipe is in PATH via home.packages, the hook just works.

        devShells.default = pkgs.mkShell {
          buildInputs = [
            pkgs.go
            pkgs.gopls
            pkgs.gotools
            pkgs.go-tools      # staticcheck
            pkgs.sshfs-fuse    # sshfs (Linux); on macOS install macFUSE manually
            pkgs.openssh
          ] ++ pkgs.lib.optionals isMacOS [
            # macFUSE cannot be installed via Nix (kernel extension).
            # Install from https://osxfuse.github.io then: brew install sshfs
          ];

          shellHook = ''
            echo "ashpipe dev shell ready"
            echo "  go build -o ashpipe . && ./ashpipe --help"
            ${pkgs.lib.optionalString isMacOS ''
              if ! command -v sshfs &>/dev/null; then
                echo ""
                echo "WARNING: sshfs not found on macOS."
                echo "  1. Install macFUSE from https://osxfuse.github.io"
                echo "  2. Run: brew install sshfs"
              fi
            ''}
          '';
        };
      });
}
