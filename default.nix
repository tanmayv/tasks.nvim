{
  pkgs ? (
    let
      inherit (builtins) fetchTree fromJSON readFile;
      inherit ((fromJSON (readFile ./flake.lock)).nodes) nixpkgs gomod2nix;
    in
    import (fetchTree nixpkgs.locked) {
      overlays = [
        (import "${fetchTree gomod2nix.locked}/overlay.nix")
      ];
    }
  ),
  buildGoApplication ? pkgs.buildGoApplication,
}:

buildGoApplication {
  pname = "task";
  version = "0.1";
  pwd = ./tui;
  src = ./tui;
  modules = ./gomod2nix.toml;
  postInstall = ''
    mv $out/bin/tui $out/bin/task
  '';
}
