self:
{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.services.rockpload;
in
{
  options.services.rockpload = {
    enable = lib.mkEnableOption "Rockpload, a Rocket League replay upload tray app";

    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
      defaultText = lib.literalExpression "inputs.rockpload.packages.\${pkgs.stdenv.hostPlatform.system}.default";
      description = "The Rockpload package to run.";
    };
  };

  config = lib.mkIf cfg.enable {
    home.packages = [ cfg.package ];

    systemd.user.services.rockpload = {
      Unit = {
        Description = "Rockpload Rocket League replay uploader";
        Requires = [ "graphical-session.target" ];
        PartOf = [ "graphical-session.target" ];
        After = [ "graphical-session.target" ];
      };

      Service = {
        ExecStart = lib.getExe cfg.package;
        Restart = "on-failure";
        RestartSec = "5s";
      };

      Install = {
        WantedBy = [ "graphical-session.target" ];
      };
    };
  };
}
