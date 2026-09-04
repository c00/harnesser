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

```

## Add Tools

By simply adding yaml files

## TODO 

- Add tools like read files, edit file, list directory, run command
- Create bubbletea textarea input

### Longer term

- We probably want to add a jsonrpc output for it too.
