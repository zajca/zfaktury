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

        # Toolchain with musl target for static CI builds
        rustToolchainMusl = pkgs.rust-bin.beta.latest.default.override {
          extensions = [ "rust-src" ];
          targets = [ "x86_64-unknown-linux-musl" ];
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
          libx11
          libxcb
          libxcomposite
          libxdamage
          libxext
          libxfixes
          libxrandr

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

        # CI devshell for static musl builds (Linux x86_64)
        # Uses pkgsStatic (musl-based, produces .a static libs) for all linked C libraries
        devShells.ci = let
          s = pkgs.pkgsStatic;
          crossCC = s.stdenv.cc;

          # Static musl libraries needed at link time
          staticLibs = [
            s.fontconfig
            s.freetype
            s.libxcb
            s.libxkbcommon
            s.zstd
            s.zlib
            s.expat
            s.libpng
            s.bzip2
            s.brotli
            s.libffi
            s.pcre2
            s.util-linux  # libuuid
            s.glib
            s.alsa-lib
            s.wayland
            s.libx11
            s.xorgproto
            s.libxau
            s.libxdmcp
          ];

          # Build library search path and pkg-config path from static packages
          mkLibPath = libs: pkgs.lib.concatMapStringsSep ":" (pkg:
            let lib = pkg.lib or pkg; in "${lib}/lib"
          ) libs;
          mkPkgConfigPath = libs: pkgs.lib.concatMapStringsSep ":" (pkg:
            let dev = pkg.dev or pkg; in "${dev}/lib/pkgconfig"
          ) libs;
          # Build -L flags for RUSTFLAGS (some -sys crates don't use pkg-config)
          mkRustLinkFlags = libs: pkgs.lib.concatMapStringsSep " " (pkg:
            let lib = pkg.lib or pkg; in "-L native=${lib}/lib"
          ) libs;
        in pkgs.mkShell {
          nativeBuildInputs = with pkgs; [
            rustToolchainMusl
            pkg-config
            cmake
            perl
            clang
            mold
          ];

          # Static musl libraries in buildInputs so Nix pkg-config hook finds their .pc files
          buildInputs = staticLibs;

          env = {
            ZSTD_SYS_USE_PKG_CONFIG = "true";
            # Use musl cross-compiler (by path to avoid polluting host includes)
            CC_x86_64_unknown_linux_musl = "${crossCC}/bin/x86_64-unknown-linux-musl-cc";
            CARGO_TARGET_X86_64_UNKNOWN_LINUX_MUSL_LINKER = "${crossCC}/bin/x86_64-unknown-linux-musl-cc";
            PKG_CONFIG_ALLOW_CROSS = "1";
            PKG_CONFIG_ALL_STATIC = "1";
            # -L flags for libraries not found via pkg-config, plus extra -l for xcb extensions
            CARGO_TARGET_X86_64_UNKNOWN_LINUX_MUSL_RUSTFLAGS = "${mkRustLinkFlags staticLibs} -l static=xcb-xkb -l static=xcb-render -l static=xcb-shape -l static=xcb-xfixes -l static=xcb-randr -l static=xcb-shm -l static=xcb-xinput -l static=Xau -l static=Xdmcp";
          };

          shellHook = ''
            echo "ZFaktury CI devshell ready (musl static, Rust $(rustc --version | cut -d' ' -f2))"
          '';
        };
      }
    );
}
