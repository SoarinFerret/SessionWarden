{
  description = "SessionWarden";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        sessionwarden = pkgs.buildGoModule {
          pname = "sessionwarden";
          version = "0.1.0";
          src = ./.;
          # List both cmd directories here:
          subPackages = [
            "cmd/sessionwardend"
            "cmd/swctl"
          ];
          vendorHash = null; # Fill this in after first build
        };
        pam_sessionwarden = pkgs.callPackage ./nix/pam-pkg.nix {};
      in {
        packages.sessionwarden = sessionwarden;
        packages.pam_sessionwarden = pam_sessionwarden;

        nixosConfigurations.vm = nixpkgs.lib.nixosSystem {
          system = "x86_64-linux";
          modules = [
            ({ config, pkgs, ... }: {
              environment.systemPackages = [
                sessionwarden
                pam_sessionwarden
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

              # Enable dconf for GNOME extension management
              programs.dconf.enable = true;
              virtualisation.vmVariant = {
                # following configuration is added only when building VM with build-vm
                virtualisation = {
                  memorySize = 4096;
                  cores = 4;
                  # Forward SSH port: host port 2222 -> guest port 22
                  forwardPorts = [
                    { from = "host"; host.port = 2222; guest.port = 22; }
                  ];
                };
              };
              services.openssh.enable = true;
              services.openssh.permitRootLogin = "yes";

              #### SystemD system service (runs as root)
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
                # Optionally, add environment variables or dependencies here
              };

              #### SystemD user service (runs per-user for notifications)
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

              ##### PAM Configuration for SessionWarden #####
              # Symlink the PAM module into the correct place
              systemd.tmpfiles.rules = [
                "L+ /lib/security/pam_sessionwarden.so - - - - ${pam_sessionwarden}/lib/security/pam_sessionwarden.so"
              ];

              # Add to the login PAM stack
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

              # write file to /etc/sessionwarden/config.toml
              environment.etc."sessionwarden/config.toml".text = ''
                [default]
                daily_limit = "2h"
                allowed_hours = "09:00-17:00"
                weekend_hours = "10:00-14:00"
                notify_before = ["10m", "5m"]
                lock_screen = true
                enabled = true

                [users]
                [users.nixos]
                daily_limit = "3h"
              '';

              services.dbus.packages = [
                (pkgs.writeTextDir "share/dbus-1/system.d/io.github.soarinferret.sessionwarden.conf" ''
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
                '')
              ];
            })
          ];
        };
      }
    );
}
