# xash-admin

A command-line RCON administration tool for [Xash3D-FWGS](https://github.com/FWGS/xash3d-fwgs) servers.

## Features

- Interactive readline interface with tab completion
- Command history, persisted to disk via XDG data directory
- Quake colour code rendering in terminal output
- Non-interactive mode for scripting and piped input
- In-band password management without shell history exposure

## Source

The canonical repository is at <https://git.vs49688.net/zane/xash-admin>.  
Any other hosting (GitHub, etc.) is a mirror.

GitHub releases are provided for convenience.

## Installation

```sh
go install git.vs49688.net/zane/xash-admin@latest
```

Or build from source:

```sh
git clone https://git.vs49688.net/zane/xash-admin
cd xash-admin
go build .
```

## Usage

```
NAME:
   xash-admin - xash-admin, a Xash3D-FWGS RCON CLI

USAGE:
   xash-admin [global options] [command [command options]]

COMMANDS:
   version  display version information
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --address string       server address (default: "127.0.0.1:27015") [$XASH_ADMIN_ADDRESS]
   --password string      rcon password [$XASH_ADMIN_PASSWORD]
   --history-file string  command history file, empty to disable, unset to use XDG default [$XASH_ADMIN_HISTORY_FILE]
   --help, -h             show help
```

### Example

```sh
xash-admin --address 192.168.1.10:27015 --password supersecret
```

Once connected, type any RCON command at the prompt:

```
(xash-admin) $ status
(server) map: dm_bulk
(server) # score ping dev  lastmsg qport useragent              name            address
(server)  0     0 Bot       n/a 51992.99065     0 n/a (n/a-n/a 0)       BadBadBot        0.0.0.0
...
(xash-admin) $ changelevel crossfire
```

## Built-in Commands

These are handled locally and are not sent to the server.

| Command                   | Description                                                         |
|---------------------------|---------------------------------------------------------------------|
| `!setpassword [password]` | Set the RCON password. Prompts securely if no argument given.       |
| `!sleep <duration>`       | Sleep for a duration (e.g. `1s`, `500ms`, `2m`). Useful in scripts. |
| `!exit`                   | Exit xash-admin.                                                    |
| `!!exit`                  | Send `exit` to the server, shutting it down.                        |
| `exit`                    | Refused — disambiguated from `!!exit` intentionally.                |

## Configuration

All options may be set via environment variables:

| Variable                  | Description                                           |
|---------------------------|-------------------------------------------------------|
| `XASH_ADMIN_ADDRESS`      | Server address, e.g. `192.168.1.10:27015`             |
| `XASH_ADMIN_PASSWORD`     | RCON password                                         |
| `XASH_ADMIN_HISTORY_FILE` | Path to history file. Set to empty string to disable. |

History is saved to `$XDG_DATA_HOME/xash-admin/history.tmp` by default.

## Non-interactive Mode

When stdout or stdin is not a terminal, xash-admin suppresses the header, prompt,
and colour codes. This makes it suitable for use in scripts:

```sh
$ printf 'maps *\n!sleep 0.1s' | ./xash-admin --address 127.0.0.1:27020 --password "$(cat /run/secrets/rcon_password)"
        dm_dust2 (Half-Life)   
    dm_macerator (Half-Life)  ZHLT v3.4 VL34 (Jul 11 2024) J.A.C.K. 1.1.3773 (vpHalfLife)
Mod_TestBmodelLumps: 2 warning(s)
         usa_box (Half-Life)   
-------------------
Directory: "mymod/maps" - Maps listed: 3
```

## License

GNU General Public License, version 2 only. See [COPYING](COPYING).

Copyright (C) 2026 Zane van Iperen &lt;zane at zanevaniperen.com&gt;
