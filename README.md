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
omnivore --hosts web1,web2,web3 -c 'uptime'
omnivore --hosts web1,web2 -c 'df -h /' -b bastion.example.com -u deploy
omnivore --hosts web1,web2 -c 'cat /etc/os-release' --no-ui   # print groups and exit
```

Hosts may include a port (`web1:2222`). Host keys are checked against `~/.ssh/known_hosts` unless
`--insecure` is given. Authentication uses, in order: the SSH agent (`SSH_AUTH_SOCK`), the key given
with `-k` (or `~/.ssh/id_ed25519`, `id_ecdsa` and `id_rsa` if present), and a password from
`OMNIVORE_PASSWORD` or `-p`. Run `omnivore --help` for every flag.

In the interface, use the arrow keys to page through individual hosts, `s` to jump to a host, `esc`
to go back to the grouped output, `CTRL+L` to show the log, and `CTRL+C` to quit.

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
- Ability to retry failed hosts a configurable number of times.

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
| `internal/ui` | The gocui interface (a read-only view of the state) and the `--no-ui` summary. |

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


