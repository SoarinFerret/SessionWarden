{ pkgs ? import <nixpkgs> {} }:

pkgs.stdenv.mkDerivation {
  pname = "pam_sessionwarden";
  version = "0.1";

  src = ../pam/pam_sessionwarden.c;
  dontUnpack = true;

  buildInputs = [
    pkgs.pam
    pkgs.dbus
  ];
   nativeBuildInputs = [ pkgs.pkg-config ];


  # Output should be a .so file for PAM
  buildPhase = ''
    mkdir -p $out/lib/security
    ${pkgs.gcc}/bin/gcc -fPIC -shared -o $out/lib/security/pam_sessionwarden.so $src \
      $(pkg-config --cflags --libs dbus-1) \
      -I${pkgs.pam}/include/security \
      -L${pkgs.pam}/lib -lpam
  '';

  installPhase = ''
    # Already installed to $out/lib/security in buildPhase
    true
  '';

  meta = with pkgs.lib; {
    homepage = "https://github.com/soarinferret/sessionwarden";
    description = "A PAM module for allowing authentication based on policy decisions made by SessionWarden";
    license = licenses.gpl3;
    platforms = platforms.unix;
  };
}
