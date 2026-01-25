{
  description = "SessionWarden - Linux session management with time-based restrictions";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    let
      # NixOS Module
      sessionwardenModule = { config, lib, pkgs, ... }:
        let
          cfg = config.services.sessionwarden;

          # Build packages with the correct system
          sessionwarden = pkgs.buildGoModule {
            pname = "sessionwarden";
            version = "0.1.0";
            src = self;
            subPackages = [
              "cmd/sessionwardend"
              "cmd/swctl"
            ];
            vendorHash = null;
          };

          pam_sessionwarden = pkgs.callPackage ./nix/pam-pkg.nix {};

          dbusPolicy = pkgs.writeTextDir "share/dbus-1/system.d/io.github.soarinferret.sessionwarden.conf" ''
            <!DOCTYPE busconfig PUBLIC
             "-//freedesktop//DTD D-BUS Bus Configuration 1.0//EN"
             "http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd">

            <busconfig>
              <!-- SessionWarden system service policy -->

              <policy user="root">
                <allow own="io.github.soarinferret.sessionwarden"/>
                <allow send_destination="io.github.soarinferret.sessionwarden"/>
                <allow receive_sender="io.github.soarinferret.sessionwarden"/>
              </policy>

              <policy user="sessionwarden">
                <allow own="io.github.soarinferret.sessionwarden"/>
                <allow send_destination="io.github.soarinferret.sessionwarden"/>
                <allow receive_sender="io.github.soarinferret.sessionwarden"/>
              </policy>

              <policy at_console="true">
                <allow send_destination="io.github.soarinferret.sessionwarden"/>
              </policy>

              <policy context="default">
                <allow receive_sender="io.github.soarinferret.sessionwarden"/>
              </policy>

              <policy context="default">
                <deny own="io.github.soarinferret.sessionwarden"/>
                <deny send_destination="io.github.soarinferret.sessionwarden"/>
              </policy>
            </busconfig>
          '';
        in {
          options.services.sessionwarden = {
            enable = lib.mkEnableOption "SessionWarden session management daemon";

            config = lib.mkOption {
              type = lib.types.lines;
              default = ''
                [default]
                daily_limit = "2h"
                allowed_hours = "09:00-20:00"
                weekend_hours = "9:00-22:00"
                notify_before = ["10m", "5m"]
                lock_screen = true
                enabled = false
              '';
              description = ''
                SessionWarden configuration in TOML format.
                This will be written to /etc/sessionwarden/config.toml
              '';
            };
          };

          config = lib.mkIf cfg.enable {
            # Add packages to system
            environment.systemPackages = [
              sessionwarden
            ];

            # SystemD system service (runs as root)
            systemd.services.sessionwardend = {
              description = "SessionWarden Daemon";
              wantedBy = [ "multi-user.target" ];
              after = [ "network.target" "dbus.service" ];
              serviceConfig = {
                Type = "simple";
                ExecStart = "${sessionwarden}/bin/sessionwardend";
                User = "root";
                Restart = "on-failure";
              };
            };

            # SystemD user service (runs per-user for notifications)
            systemd.user.services.sessionwardend-user = {
              description = "SessionWarden User Notification Listener";
              wantedBy = [ "default.target" ];
              after = [ "graphical-session.target" ];
              partOf = [ "graphical-session.target" ];
              serviceConfig = {
                Type = "simple";
                ExecStart = "${sessionwarden}/bin/sessionwardend --user";
                Restart = "on-failure";
                RestartSec = "5s";
              };
            };

            # PAM Configuration
            systemd.tmpfiles.rules = [
              "L+ /lib/security/pam_sessionwarden.so - - - - ${pam_sessionwarden}/lib/security/pam_sessionwarden.so"
            ];

            security.pam.services.login.rules.account.sessionwarden = {
              enable = true;
              order = config.security.pam.services.login.rules.account.unix.order - 10;
              control = "required";
              modulePath = "/lib/security/pam_sessionwarden.so";
            };

            security.pam.services.login.rules.auth.sessionwarden = {
              enable = true;
              order = config.security.pam.services.login.rules.auth.unix.order - 400;
              control = "required";
              modulePath = "/lib/security/pam_sessionwarden.so";
            };

            # Configuration file
            environment.etc."sessionwarden/config.toml".text = cfg.config;

            # D-Bus policy
            services.dbus.packages = [ dbusPolicy ];
          };
        };
    in
    {
      # Export the NixOS module
      nixosModules.default = sessionwardenModule;
      nixosModules.sessionwarden = sessionwardenModule;

      # VM configuration for testing
      nixosConfigurations.vm = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        modules = [
          sessionwardenModule
          ({ config, pkgs, ... }: {
            # Enable SessionWarden for VM testing
            services.sessionwarden = {
              enable = true;
              config = ''
                [default]
                daily_limit = "2h"
                allowed_hours = "09:00-17:00"
                weekend_hours = "10:00-14:00"
                notify_before = ["10m", "5m"]
                lock_screen = true
                enabled = false

                [users]
                [users.nixos]
                enabled = true
                daily_limit = "3h"
              '';
            };

            # VM-specific configuration
            environment.systemPackages = [
              pkgs.gnomeExtensions.fullscreen-notifications
            ];

            services.xserver.enable = true;
            services.xserver.desktopManager.gnome.enable = true;
            services.xserver.desktopManager.gnome.extraGSettingsOverrides = ''
              [org.gnome.shell]
              enabled-extensions=['fullscreen-notifications@sorrow.about.alice.pm.me']
            '';
            services.xserver.displayManager.gdm.enable = true;
            services.xserver.displayManager.gdm.wayland = true;

            time.timeZone = "America/Chicago";

            users.users.root.password = "nixos";
            users.users.nixos = {
              isNormalUser = true;
              password = "nixos";
            };

            programs.dconf.enable = true;

            virtualisation.vmVariant = {
              virtualisation = {
                memorySize = 4096;
                cores = 4;
                forwardPorts = [
                  { from = "host"; host.port = 2222; guest.port = 22; }
                ];
              };
            };

            services.openssh.enable = true;
            services.openssh.permitRootLogin = "yes";
          })
        ];
      };
    } // flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        sessionwarden = pkgs.buildGoModule {
          pname = "sessionwarden";
          version = "0.1.0";
          src = ./.;
          subPackages = [
            "cmd/sessionwardend"
            "cmd/swctl"
          ];
          vendorHash = null;
        };
        pam_sessionwarden = pkgs.callPackage ./nix/pam-pkg.nix {};
      in {
        packages.default = sessionwarden;
        packages.sessionwarden = sessionwarden;
        packages.pam_sessionwarden = pam_sessionwarden;
      }
    );
}
