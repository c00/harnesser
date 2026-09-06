package replacetextcmd

var ExampleYaml = `enabled: true
trusted: false
tool:
  name: "replace-text"
  description: "Replace literal text in a file if the number of matches is as expected."
  parameters:
    type: object
    properties:
      filename:
        type: string
        description: "The file to edit."
      search:
        type: string
        description: "The literal text to search for."
      replacement:
        type: string
        description: "The text that replaces every match."
      expectedMatches:
        type: number
        description: "The number of matches required before the file is edited."
        default: 1
    required:
      - filename
      - search
      - replacement
command:
  - '{{ with index .Env "HARNESSER_BIN"}}{{.}}{{else}}harnesser{{end}}'
  - "tool"
  - "replace-text"
  - "{{ .Params.filename }}"
  - "{{ .Params.search }}"
  - "{{ .Params.replacement }}"
  - '{{ if ne (index .Params "expectedMatches") nil }}--expected-matches={{ index .Params "expectedMatches" }}{{ end }}'
`
