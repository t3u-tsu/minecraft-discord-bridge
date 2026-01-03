{
  description = "Minecraft Discord Bridge in Go";

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
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            sqlite
            gcc # For go-sqlite3 (CGO)
          ];

          shellHook = ''
            echo "Minecraft Discord Bridge Development Environment"
            export CGO_ENABLED=1
          '';
        };
      }
    );
}
