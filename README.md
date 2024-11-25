# mm-channel-users

`mm-channel-users` is a utility designed to generate a list of users within public channels for a given team, within a Mattermost instance. It is intended for Mattermost sysadmins and integrates with the Mattermost API to retrieve and process this information.

## Getting Started

To use `mm-channel-users`, download the appropriate executable for your platform from the [Releases](https://github.com/jlandells/mm-channel-users/releases) page
Make sure to place the downloaded executable in your `PATH` or reference it directly when running commands.

## Usage

### Basic Command

To run `mm-channel-users`, you must specify the Mattermost server URL and API token, along with the username for which the channel count should be generated:

```bash
mm-channel-users -url=mattermost.example.com -scheme=https -port=443 -token=your_api_token -username=sample.user -team=your_selected_team_name
```

### Command Line Parameters

| **Command Line** | **Environment** | **Notes** |
| --- | --- | --- |
| `-url` | `MM_URL` | **Required**. The Mattermost host that will receive the API requests. |
| `-scheme` | `MM_SCHEME` | `http` / `https`. Defaults to `http`. |
| `-port` | `MM_PORT` | The port used to reach the Mattermost instance. Defaults to `8065`. |
| `-token` | `MM_TOKEN` | **Required**. The API token used to access Mattermost. The user **must** have sysadmin rights. |
| `-team` |  | **Required**. The Mattermost team against which this utility should run. |
| `-type` |  | The type of file (CSV or JSON) to be generated.  Defaults to `CSV`. |
| `-file` |  | The filename for the output.  If no filename is specified, it will be written to `stdout`. |
| `-includebots` |  | By default, the output only contains real, active users.  By supplying this flag, bot accounts will also be included. |
| `-debug` | `MM_DEBUG` | Executes the application in debug mode, providing additional output. |
| `-version` |  | Prints the current version and exits. |
| `-help` |  | Displays usage instructions and exits. |

### Examples

**Creating a CSV file**

```bash
mm-channel-users -url=mattermost.example.com -token=your_api_token -team=sample-team -file=sample.csv
```

**Using environment variables, JSON output, and including bots:**

```bash
export MM_URL=mattermost.example.com
export MM_TOKEN=your_api_token
mm-channel-users -team=sample-team -type=JSON -file=sample.json -includebots
```

**Enabling debug mode:**

```bash
mm-channel-users -url=mattermost.example.com -token=your_api_token -team=sample-team -debug=true
```

In all examples, command-line parameters will override corresponding environment variables.

## Contributing

We welcome contributions from the community! Whether it's a bug report, a feature suggestion, or a pull request, your input is valuable to us. Please feel free to contribute in the following ways:
- **Issues and Pull Requests**: For specific questions, issues, or suggestions for improvements, open an issue or a pull request in this repository.
- **Mattermost Community**: Join the discussion in the [Integrations and Apps](https://community.mattermost.com/core/channels/integrations) channel on the Mattermost Community server.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contact

For questions, feedback, or contributions regarding this project, please use the following methods:
- **Issues and Pull Requests**: For specific questions, issues, or suggestions for improvements, feel free to open an issue or a pull request in this repository.
- **Mattermost Community**: Join us in the Mattermost Community server, where we discuss all things related to extending Mattermost. You can find me in the channel [Integrations and Apps](https://community.mattermost.com/core/channels/integrations).
- **Social Media**: Follow and message me on Twitter, where I'm [@jlandells](https://twitter.com/jlandells).

