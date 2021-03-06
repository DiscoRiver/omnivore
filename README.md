![Designed by dgim-studio / Freepik](https://github.com/DiscoRiver/omnivore/blob/main/asset/omnivore_logo.png)
Logo designed by dgim-studio

Omnivore is an interactive distributed SSH tool. Designed to quickly identify output anomalies and inconsistencies in large environments.

This program was developed alongside https://github.com/discoriver/massh, and is tightly integrated with it. It's my hope that Omnivore not only serves as a useful tool, but also as a demonstration of how one might utilise `Massh` elegantly to provide powerful programs that require advanced SSH output processing.

## Example Usage (including host panels and search)
https://user-images.githubusercontent.com/25288774/177227850-5436ff29-4859-4c55-aa56-f555deb2e4c9.mov

## Upcoming features
- Generally more interactive ability from within the program itself. Running new commands on the same host set, changing public key used, viewing output for individual hosts in their own views etc.
- Ability for scripts to be executed.
- Ability to retry failed hosts a configurable number of times.
- Better SSH key handling.
- Omnivore will eventually offload in-memory storage of host output to files within `~/.omnivore/history/`, however we need to build a system for preserving in-memory functionality as a fallback if writing fails, to avoid crashing completely.

## Known Issues

- If using iTerm2 on MacOS, you will need to edit your profile, and under the `Terminal` heading, uncheck the option `Save lines to scrollback in alternate screen mode`. Leaving this option checked will cause major trauma to your scroll buffer when resizing the terminal window when Omnivore is running.

## Noteworthy

- Application logs are located at `~/.omnivore/history/`. There is a timestamped directory for each run of omnivore, containing a file for each host containing it's output, and a log file for that run. 

## Testing

Testing group should be appended to all tests, either `UnitWorkflow` or `IntegrationWorkflow`.

Running a group can be done with the `-run` flag;

* `go test -v ./... -run UnitWorkflow`
* `go test -v ./... -run IntegrationWorkflow`

For integration testing, which performs an SSH "hello world", Omnivore uses cluster of machines within an AWS EC2 group "omnivore". Locally, you can configure your own AWS credentials that has access to such a group and it will run against whichever hosts it finds. As long as you have credentials configured for AWS EC2, permission, and a group with the applicable name, it should run without issue. 

## Acknowledgements

- Dalal, for enriching my life and keeping me on the rails.
- Tim for putting his trust in me to do good work.
- Adam & Bryan for helping me see the value.
- Alysha for always propping up my ego.


