{ pkgs ? import <nixpkgs> {} }:

pkgs.stdenv.mkDerivation {
  pname = "pam_sessionwarden";
  version = "0.1";

  src = ../pam/pam_sessionwarden.c;

  buildInputs = [
    pkgs.pam
    pkgs.dbus
  ];

  # Output should be a .so file for PAM
  buildPhase = ''
    mkdir -p $out/lib/security
    ${pkgs.gcc}/bin/gcc -fPIC -shared -o $out/lib/security/pam_sessionwarden.so $src \
      -I${pkgs.pam.dev}/include/security \
      -I${pkgs.dbus.dev}/include/dbus-1.0 \
      -I${pkgs.dbus.dev}/lib/dbus-1.0/include \
      -L${pkgs.pam.out}/lib -lpam \
      -L${pkgs.dbus.out}/lib -ldbus-1
  '';

  installPhase = ''
    # Already installed to $out/lib/security in buildPhase
    true
  '';

  meta = with lib; {
    homepage = "https://github.com/soarinferret/sessionwarden";
    description = "A PAM module for allowing authentication based on policy decisions made by SessionWarden";
    license = licenses.gpl3;
    platforms = platforms.unix;
  };
}
