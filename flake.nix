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
      in {
        packages.sessionwarden = sessionwarden;

        nixosConfigurations.vm = nixpkgs.lib.nixosSystem {
          system = "x86_64-linux";
          modules = [
            ({ pkgs, ... }: {
              environment.systemPackages = [ sessionwarden ];
              services.xserver.enable = true;
              services.xserver.desktopManager.gnome.enable = true;
              services.xserver.displayManager.gdm.enable = true;
              services.xserver.displayManager.gdm.wayland = true;
              users.users.root.password = "nixos";
              users.users.nixos = {
                isNormalUser = true;
                password = "nixos";
              };
              virtualisation.vmVariant = {
                # following configuration is added only when building VM with build-vm
                virtualisation = {
                  memorySize = 4096;
                  cores = 4;
                };
              };
              services.openssh.enable = true;
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
