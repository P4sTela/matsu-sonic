{
  description = "gdrive-sync development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-tools
            bun
            sqlite
            gcc
            pkgsCross.mingwW64.stdenv.cc
            go-task
            tygo
            air
          ];

          shellHook = ''
            export CGO_ENABLED=1
          '';
        };
      });
}
