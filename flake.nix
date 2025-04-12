{
  description = "Go async library development environment";

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
            gopls
            delve
            golangci-lint
          ];

          shellHook = ''
            echo "Go development environment"
            echo "Available tools:"
            echo "  - go: Go compiler"
            echo "  - gopls: Go language server"
            echo "  - delve: Go debugger"
            echo "  - golangci-lint: Go linter"
          '';
        };
      });
} 