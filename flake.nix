{
  description = "Rockpload desktop app";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
      ];

      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
          rockploadVersion = builtins.replaceStrings [ "\n" "\r" " " "\t" ] [ "" "" "" "" ] (
            builtins.readFile ./VERSION
          );

          runtimeLibraries = with pkgs; [
            libglvnd
            libx11
            libxcursor
            libxrandr
            libxinerama
            libxi
            libxxf86vm
            mesa
          ];
        in
        {
          default = pkgs.buildGoModule {
            pname = "rockpload";
            version = rockploadVersion;

            src = self;
            vendorHash = null;

            go = pkgs.go_1_26;

            nativeBuildInputs = with pkgs; [
              makeWrapper
              pkg-config
            ];

            buildInputs = runtimeLibraries;

            ldflags = [
              "-s"
              "-w"
              "-X main.Version=${rockploadVersion}"
            ];

            postInstall = ''
              wrapProgram "$out/bin/rockpload" \
                --prefix LD_LIBRARY_PATH : "${pkgs.lib.makeLibraryPath runtimeLibraries}"
            '';

            meta = {
              description = "Desktop companion app for automatically uploading Rocket League replays";
              homepage = "https://github.com/LEX0RE/rockpload";
              license = pkgs.lib.licenses.asl20;
              mainProgram = "rockpload";
              platforms = pkgs.lib.platforms.linux;
            };
          };
        }
      );

      nixosModules.default = import ./nix/nixos-module.nix self;
      nixosModules.rockpload = self.nixosModules.default;

      homeManagerModules.default = import ./nix/home-manager-module.nix self;
      homeManagerModules.rockpload = self.homeManagerModules.default;

      apps = forAllSystems (system: {
        default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/rockpload";
          meta.description = "Run Rockpload";
        };
      });

      devShells = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };

          runtimeLibraries = with pkgs; [
            libglvnd
            libx11
            libxcursor
            libxrandr
            libxinerama
            libxi
            libxxf86vm
            mesa
          ];
        in
        {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go_1_26
              gopls
              go-tools
              gotools
              pkg-config
              stdenv.cc
              pkgsCross.mingwW64.stdenv.cc.cc
            ];

            inputsFrom = [
              self.packages.${system}.default
            ];

            LD_LIBRARY_PATH = pkgs.lib.makeLibraryPath runtimeLibraries;
            CGO_ENABLED = "1";
          };
        }
      );

      formatter = forAllSystems (system: nixpkgs.legacyPackages.${system}.nixfmt);
    };
}
