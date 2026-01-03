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
            gcc
          ];

          shellHook = '
            echo "Minecraft Discord Bridge Development Environment"
            export CGO_ENABLED=1
          ';
        };

        packages.default = pkgs.buildGoModule {
          pname = "minecraft-discord-bridge";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-W1qCmaJkLVEfBlxvIvsGhui84HOUHcKi+boC0lvozOo=";

          buildInputs = [ pkgs.sqlite ];
          env.CGO_ENABLED = 1;
        };
      }
    );
}
