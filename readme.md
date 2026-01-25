# SessionWarden

SessionWarden is a session management system for Linux. It allows administrators to limit session durations, enforce automatic logouts, and more. SessionWarden was designed by a frustrated system administrator (dad) in need of better session control tools for his users (kids).

## Disclaimer

This project has the potential to lock you out of your system if misconfigured (or from bugs). Please use caution when installing and configuring SessionWarden, especially on production systems. This software is provided "as is", without warranty of any kind. By using this software, you agree that you are solely responsible for any consequences that may arise from its use.

Additionally, AI (GitHub Copilot) was used to help write and sketch out parts of this project, since I don't have prior development experience with PAM modules and D-Bus (though really good/fun learning experience!).

## Features

* Simple config file for easy setup
* Daily session limits - set maximum amount of time a user can be logged in per day
  * Example: limit user "bob" to 2 hours of session time per day
  * Automatic logout when limit is reached, including optional notifications
* Session tracking - monitor active sessions and their durations
* Login time restrictions - restrict login times for specific users
  * Example: allow user "alice" to log in only between 4 PM and 8 PM
* Override options for administrators
  * Example: add extra time to a user's session limit in case of special circumstances
* Notifications - notify users before their session limit is reached
* CLI tool for administrators to manage and monitor sessions
  * Send custom notifications to users
  * View current session statuses
  * Manage overrides and session states
  * Pause and resume user sessions

## Design

SessionWarden has a PAM (Pluggable Authentication Module), system background daemon, and a user background daemon. The PAM module intercepts user logins and logouts to track session durations and enforces restrictions. The system background daemon does all the heavy lifting, while the user background daemon just receives notifications from the system daemon and displays them to the user. A CLI tool is provided for administrators to manage configurations and view session statistics.

* `sessionwardend` - the background daemon
  * `sessionwardend --user` - the user notification daemon
* `pam_sessionwarden.so` - the PAM module
* `swctl` - command line interface for managing SessionWarden

### Configuration Files / Data Storage

* `/etc/sessionwarden/config.toml` - main configuration file
* `/var/lib/sessionwarden/state.json` - current session state, usage data, and overrides
* `/var/log/sessionwarden/sessionwarden.log` - log file for SessionWarden activities

### Configuration Options

```toml
[defaults]
daily_limit = "2h"
allowed_hours = "08:00-20:00"
notify_before = ["15m","5m"]
lock_screen = true # lock screen when limit is reached instead of logging out
enabled = false

[users.alice]
enabled = true
daily_limit = "unlimited"

[users.bob]
enabled = true
daily_limit = "3h"
```

## CLI Usage

```
$ swctl --help
swctl allows you to interact with the SessionWarden service via D-Bus.
			You can use it to query session status, manage sessions, and more.

Usage:
  swctl [flags]
  swctl [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  notify      Send a notification to a user
  override    Manage temporary policy overrides
  pause       Pause / lock user session until manually resumed
  ping        Check if SessionWarden daemon is running
  resume      Resume session for a user
  user        Show detailed status for a user

Flags:
  -h, --help   help for swctl

Use "swctl [command] --help" for more information about a command.
```

## NixOS Integration

Included is a NixOS module for easy integration with NixOS systems.

Add this to your NixOS configuration:

```nix
 {
   inputs = {
     nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
     sessionwarden.url = "github:SoarinFerret/SessionWarden";
   };

   outputs = { nixpkgs, sessionwarden, ... }: {
     nixosConfigurations.yourhostname = nixpkgs.lib.nixosSystem {
       system = "x86_64-linux";
       modules = [
         sessionwarden.nixosModules.default
         ./configuration.nix
       ];
     };
   };
 }
```

Then in your configuration.nix:

```nix
 {
   # Enable SessionWarden
   services.sessionwarden = {
     enable = true;
     config = ''
       [default]
       daily_limit = "4h"
       allowed_hours = "08:00-22:00"
       weekend_hours = "08:00-22:00"
       notify_before = ["15m", "5m"]
       lock_screen = true
       enabled = true

       [users]
       [users.alice]
       daily_limit = "6h"
       enabled = true
     '';
   };
 }
```

## License

SessionWarden is licensed under the GPL-3.0 License. See the LICENSE file for more details.
