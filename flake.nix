{
  description = "Worktree Manager - A CLI tool for managing git worktrees";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages = {
          default = pkgs.callPackage ./package.nix { };
          wtm = pkgs.callPackage ./package.nix { };
        };

        apps = {
          default = {
            type = "app";
            program = "${self.packages.${system}.default}/bin/wtm";
          };
          wtm = {
            type = "app";
            program = "${self.packages.${system}.wtm}/bin/wtm";
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gotools
            gopls
            git
          ];

          shellHook = ''
            echo "Worktree Manager development environment"
            echo "Go version: $(go version)"
            echo "Git version: $(git --version)"
          '';
        };
      }
    );
}
