package readfilecmd

var ExampleYaml = `enabled: true
trusted: false
tool:
  name: "read-file"
  description: "Read a file from disk. You can limit the range to a number of lines and bytes."
  parameters:
    type: object
    properties:
      filename:
        type: string
        description: "The file to read."
      startLine:
        type: number
        description: "The line to start reading at."
        default: 1
      maxLines:
        type: number
        description: "The maximum lines returned."
        default: 100
      maxBytes:
        type: number
        description: "The maximum bytes returned."
        default: 1024
      includeLineNumbers:
        type: boolean
        description: "If true, line numbers are included in the output."
        default: false
    required:
      - filename
command:
  - '{{ with index .Env "HARNESSER_BIN"}}{{.}}{{else}}harnesser{{end}}'
  - "tool"
  - "read-file"
  - "{{ .Params.filename }}"
  - '{{ with index .Params "startLine" }}--start={{ . }}{{ end }}'
  - '{{ with index .Params "maxLines" }}--max-lines={{ . }}{{ end }}'
  - '{{ with index .Params "maxBytes" }}--max-bytes={{ . }}{{ end }}'
  - '{{ with index .Params "includeLineNumbers" }}--line-numbers{{ end }}'
`
