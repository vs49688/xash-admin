{
  description = "xash-admin";

  inputs = {
    nixpkgs.url = github:NixOS/nixpkgs;
  };

  outputs = { self, nixpkgs, ... }: {
    packages.x86_64-linux = let
      pkgs = self.inputs.nixpkgs.legacyPackages.x86_64-linux;
    in {
      default = pkgs.callPackage ./default.nix {
        version = "0.0.0-${self.lastModifiedDate}-${if (self ? rev) then self.shortRev else self.dirtyShortRev}";
      };
    };
  };
}
