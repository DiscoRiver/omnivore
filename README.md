![Designed by dgim-studio / Freepik](https://github.com/DiscoRiver/omnivore/blob/main/asset/omnivore_logo.png)
Logo designed by dgim-studio

Omnivore is an interactive distributed SSH tool. Designed to quickly identify output anomalies and inconsistencies in large environments.

This program was developed alongside https://github.com/discoriver/massh, and is tightly integrated with it. It's my hope that Omnivore not only serves as a useful tool, but also as a demonstration of how one might utilise `Massh` elegantly to provide powerful programs that require advanced SSH output processing.

## Example Usage (including host panels and search)
https://user-images.githubusercontent.com/25288774/177227850-5436ff29-4859-4c55-aa56-f555deb2e4c9.mov

## Install

```
go install github.com/discoriver/omnivore@latest
```

## Usage

```
omnivore --hosts 'web[01-20]' -c 'uptime'
omnivore --hosts web1,web2 -c 'df -h /' -b bastion.example.com -u deploy
omnivore -f hosts.txt -c 'systemctl restart nginx' --sudo
omnivore --hosts web1,web2 -c 'cat /etc/os-release' --no-ui   # print groups; exit 1 if any host failed
```

- **Hosts** come from `--hosts` (comma separated or repeated) and `--hosts-file`/`-f` (one or more
  per line, `#` comments). Ranges are expanded: `web[01-03,07]` is web01, web02, web03 and web07.
  Hosts may include a port (`web1:2222`).
- **`~/.ssh/config` is honoured**: `HostName`, `Port`, `User`, `IdentityFile` and single-hop
  `ProxyJump`, so aliases work as they do with `ssh`. `-u`, `-b` and ports in `--hosts` take
  precedence. `ProxyCommand` and multi-hop jumps aren't supported. Use `--no-ssh-config` to ignore it.
- **Authentication** uses the SSH agent, then the key given with `-k` (or `~/.ssh/id_ed25519`,
  `id_ecdsa` and `id_rsa`) plus each host's `IdentityFile`s, then a password from `--ask-pass`/`-P`,
  `OMNIVORE_PASSWORD` or `-p`.
- **Host keys** are checked against `~/.ssh/known_hosts`. `--accept-new` trusts and records hosts
  seen for the first time but still rejects changed keys; `--insecure` skips checks entirely.
- **sudo**: `--sudo` prompts once for the sudo password and sends it to every host over the SSH
  connection (never on a command line), or set `OMNIVORE_SUDO_PASSWORD`. Leave it empty to use
  passwordless sudo, which then fails at once instead of hanging on a prompt.
- **Slow hosts**: `--command-timeout N` flags hosts with no output for N seconds; `--kill-slow`
  also stops them.
- **Retrying**: failed hosts are written to `failed.txt` in the run's history directory, and
  omnivore prints the `--hosts-file` argument to retry them.

Run `omnivore --help` for every flag.

### The interface

- **Groups** (the start screen) lists hosts grouped by identical output and exit status, largest
  first, so the odd ones out are at the bottom, tagged `≠ majority`. Use `↑`/`↓` to select a group,
  `enter` to open its first host, and `d` to see exactly how it differs from the majority.
- **Host view** shows one host's output, live while it runs (`f` toggles following). `←`/`→` page
  through hosts; `tab` filters them (and the host list on the left) by status.
- `/` finds a host by name as you type, `y` copies the selected group's hosts to the clipboard,
  `l` shows the log, `?` shows every key, and `q` quits (asking first if hosts are still running).
- Everything scrolls (keys or mouse wheel), and the layout adapts to the terminal size and
  light or dark backgrounds. Terminal escape sequences in remote output are stripped before display.
- In terminals that can show images (Kitty, Ghostty, Konsole, iTerm2 and WezTerm), a bear in the
  top-left corner keeps you company: it chomps while hosts run, roars when one fails, and is happy
  when everything succeeds or puzzled if anything failed. Set `OMNIVORE_MASCOT=off` to hide it, or
  `kitty`/`iterm2` to force a protocol (it's off inside tmux and screen). The frames are drawn by
  `tools/mascot/draw.py`.

### Config file

Settings can also be kept in `~/.omnivore/config.yaml` (or a file given with `--config`), which must
only be readable by you (`chmod 600`). Flags take precedence.

```yaml
omni:
  Username: deploy
  BastionHost: bastion.example.com
  PrivateKeyLoc: ~/.ssh/id_ed25519
  PrivateKeyPassword: ""
  SSHTimeout: 10      # seconds
  CommandTimeout: 30  # seconds without output before a host is flagged slow; 0 disables
  WorkerPool: 30      # hosts to run on at once
```

## Upcoming features
- Generally more interactive ability from within the program itself. Running new commands on the same host set, changing public key used etc.
- Ability for scripts to be executed.

## Known Issues

- If using iTerm2 on MacOS, you will need to edit your profile, and under the `Terminal` heading, uncheck the option `Save lines to scrollback in alternate screen mode`. Leaving this option checked will cause major trauma to your scroll buffer when resizing the terminal window when Omnivore is running.

## Noteworthy

- Each run of omnivore gets a timestamped directory in `~/.omnivore/history/`, containing a file with each host's output, and the run's log (`log.txt`).

## Development

The code is layered so that each part can be tested on its own:

| Package | Role |
|---|---|
| `internal/cli` | The cobra command: wires everything together. |
| `internal/config` | Resolves flags, config file and defaults; builds the `massh.Config`. |
| `internal/runner` | Feeds the `massh` event stream into the state and history. |
| `internal/state` | Concurrency-safe model of a run: host statuses, output and groups. |
| `internal/group` | Groups hosts by identical output. |
| `internal/store` | Run history on disk. |
| `internal/logging` | `log/slog` setup, with an in-memory buffer for the log panel. |
| `internal/ui` | The Bubble Tea interface (a read-only view of the state) and the `--no-ui` summary. |
| `internal/hosts` | Host lists: ranges like `web[01-20]` and host files. |
| `internal/sshconfig` | Reads the `~/.ssh/config` settings omnivore honours. |

Tests are hermetic: SSH tests use the in-process server from `github.com/discoriver/massh/v2/sshtest`,
so no sshd, SSH keys or cloud hosts are needed.

```
go test -race ./...
```

## Acknowledgements

- Dalal, for enriching my life and keeping me on the rails.
- Tim for putting his trust in me to do good work.
- Adam & Bryan for helping me see the value.
- Alysha for always propping up my ego.


