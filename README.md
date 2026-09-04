# Harnesser

Dead simple AI harnesses. Tools, multiple providers, agents, multi-turn conversations.

Create a new harness for every project.

VERY MUCH IN ALPHA

## Installation

```sh
# Install
go install github.com/c00/harnesser/cmd/harnesser

# Initialize config files and set api key
harnesser init --user
```

Currently it only supports openrouter. Find your config file at `~/.harnesser/config.yaml` to set-up basic llm configuration.

The api key is stored in the keyring (linux, Mac should be supported).

## Usage

```sh
harnesser start
```

## Add Tools

By simply adding yaml files to the user or local `.harnesser/tools/` folder, you can add functionality. Example:

```yaml
enabled: true # Allow this tool to be used
trusted: true # Set to false if you want to manually approve the execution of the tool
allowedEnv: # Env variables that will be interpolated in the command (See command section)
  - FIRECRAWL_API_KEY
tool:
  name: "firecrawl_search"
  description: "Search the web. Query is required."
  parameters: # The JSON schema that AI endpoints expect
    type: object
    properties:
      query:
        type: string
        description: "The search query to send to Firecrawl."
    required:
      - query
# The command to be executed. Each part of the command is interpolated to add variables from the parameters and environment into it.
# In the example below we add the environment variable FIRECRAWL_API_KEY into the authorization header
# and we add the query into the json body of the request.
# The helper function `json` is there to properly encode the value.
command: 
  - "curl"
  - "-X"
  - "POST"
  - "https://api.firecrawl.dev/v1/search"
  - "-H"
  - "Content-Type: application/json"
  - "-H"
  - "Authorization: Bearer {{ .Env.FIRECRAWL_API_KEY }}"
  - "-d"
  - '{ "query": {{ json .Params.query }} }'
```

## TODO 

- Fix sigint when reading stdin
- Add tools like read files, edit file, list directory, run command
  - Restructure the subcommands to make this better
  - Start with reading tools
- Create bubbletea textarea input
- Add support for `allowedSecrets` and `{{ .Secrets.FOO }}` in interpolation.
- Make 'start' the default if no args are given

### Longer term

- We probably want to add a jsonrpc output for it too.
- Add LSM to limit the damage this thing can do to your system.
