{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = [
    pkgs.gcc
    pkgs.pkg-config
    pkgs.gtk3
    pkgs.webkitgtk_4_1
    pkgs.libsoup_3
    pkgs.glib
  ];
}
