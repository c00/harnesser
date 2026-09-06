package interpolator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"text/template"

	"github.com/c00/harnesser/models"
	"github.com/xeipuuv/gojsonschema"
)

type ToolCallData struct {
	Env    map[string]string
	Params map[string]any
	// Potentially add more context if needed.
}

func InterpolatedCommand(tc models.ToolCall, td models.ToolDefinition) ([]string, ToolCallData, error) {
	interpolatedArgs := []string{}

	// Build params
	data := ToolCallData{
		Env: map[string]string{},
	}

	var err error

	data.Params, err = getArgsAsParams(tc.Args, td.Tool.Parameters)
	if err != nil {
		return nil, data, fmt.Errorf("cannot get toolcall args as parameters: %w", err)
	}

	for _, key := range td.AllowedEnv {
		val := os.Getenv(key)
		if val == "" {
			return nil, data, fmt.Errorf("missing env variable: %v", key)
		}
		data.Env[key] = val
	}

	// For each arg, interpolate with template/text
	for _, part := range td.Command {
		tmpl, err := template.New("argument").
			Option("missingkey=error").
			Funcs(template.FuncMap{
				"json": func(v any) (string, error) {
					b, err := json.Marshal(v)
					return string(b), err
				},
			}).
			Parse(part)

		if err != nil {
			return nil, data, fmt.Errorf("cannot parse template for part %v: %w", part, err)
		}

		var interpolated bytes.Buffer
		if err := tmpl.Execute(&interpolated, data); err != nil {
			return nil, data, fmt.Errorf("cannot execute template for part %v: %w", part, err)
		}

		interpolatedArgs = append(interpolatedArgs, interpolated.String())

	}

	return interpolatedArgs, data, nil
}

func getArgsAsParams(data string, schema map[string]any) (map[string]any, error) {
	if schema == nil {
		schema = map[string]any{}
	}

	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("cannot marshall json schema: %w", err)
	}

	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)
	dataLoader := gojsonschema.NewStringLoader(data)

	results, err := gojsonschema.Validate(schemaLoader, dataLoader)
	if err != nil {
		return nil, fmt.Errorf("cannot validate schema: %w", err)
	}

	if !results.Valid() {
		return nil, fmt.Errorf("tool call parameters not valid. schema: %v, values: %v", string(schemaBytes), data)
	}

	dataMap := map[string]any{}
	err = json.Unmarshal([]byte(data), &dataMap)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshall paramaters: %w", err)
	}

	return dataMap, nil
}
