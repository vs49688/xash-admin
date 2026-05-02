{ buildGoModule, version }:
buildGoModule(finalAttrs: {
  inherit version;

  pname = "xash-admin";

  src = ./.;

  vendorHash = null;

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s" "-w"
    "-extldflags=-static"

    "-X git.vs49688.net/zane/xash-admin/config.AppVersion=v${finalAttrs.version}"
  ];
})
