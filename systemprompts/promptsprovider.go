package systemprompts

import "github.com/c00/harnesser/types"

type PromptsReader interface {
	// Read all prompts as their own struct
	Prompts() Prompts
}

type Prompt struct {
	Active  bool
	Key     string
	Message types.Message
}

type Prompts []Prompt

func NewPrompt(key string, text string) Prompt {
	return Prompt{
		Active:  true,
		Key:     key,
		Message: types.NewSystemMessage(text),
	}
}

// ActiveMessages returns all acrive prompts as models.Message
func (p Prompts) ActiveMessages() types.Messages {
	msgs := types.Messages{}
	for _, prompt := range p {
		if prompt.Active {
			msgs = append(msgs, prompt.Message)
		}
	}
	return msgs
}
