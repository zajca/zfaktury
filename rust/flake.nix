{
  description = "ZFaktury Rust - GPUI desktop application";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    rust-overlay.url = "github:oxalica/rust-overlay";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, rust-overlay, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        overlays = [ (import rust-overlay) ];
        pkgs = import nixpkgs { inherit system overlays; };

        # GPUI requires Rust 1.93+ with edition 2024 support
        rustToolchain = pkgs.rust-bin.beta.latest.default.override {
          extensions = [ "rust-src" "rust-analyzer" ];
        };

        nativeBuildInputs = with pkgs; [
          rustToolchain
          pkg-config
          cmake
          perl
          clang
          mold
        ];

        buildInputs = with pkgs; [
          # GPU rendering (wgpu/Vulkan)
          vulkan-loader
          libdrm
          libgbm

          # Wayland
          wayland

          # X11 (fallback)
          xorg.libX11
          xorg.libxcb
          xorg.libXcomposite
          xorg.libXdamage
          xorg.libXext
          xorg.libXfixes
          xorg.libXrandr

          # Keyboard input
          libxkbcommon

          # Fonts
          fontconfig
          freetype

          # TLS / HTTP
          openssl
          curl

          # Compression
          zlib
          zstd

          # System
          glib
          alsa-lib

          # Headless GUI testing
          cage
          grim

          # Dev tools
          cargo-llvm-cov
        ];

        runtimeLibs = with pkgs; [
          vulkan-loader
          wayland
          libxkbcommon
          libva
        ];
      in
      {
        devShells.default = pkgs.mkShell {
          inherit nativeBuildInputs buildInputs;

          env = {
            ZSTD_SYS_USE_PKG_CONFIG = "true";
            LD_LIBRARY_PATH = pkgs.lib.makeLibraryPath runtimeLibs;
            FONTCONFIG_FILE = pkgs.makeFontsConf {
              fontDirectories = [
                "${pkgs.inter}/share/fonts"
                "${pkgs.jetbrains-mono}/share/fonts"
                "${pkgs.dejavu_fonts}/share/fonts"
              ];
            };
          };

          shellHook = ''
            echo "ZFaktury Rust devshell ready (Rust $(rustc --version | cut -d' ' -f2))"
          '';
        };
      }
    );
}
