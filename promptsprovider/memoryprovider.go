package promptsprovider

var _ PromptsReader = (*MemoryProvider)(nil)

// MemoryProvider provides prompts from memory. Useful for testing or other situations where you don't
// want to read prompts from disk.
type MemoryProvider struct {
	prompts []Prompt
}

func (p *MemoryProvider) Prompts() Prompts {
	return p.prompts
}

// SetPrompts replaces the existing prompts with the given prompts.
func (p *MemoryProvider) SetPrompts(prompts Prompts) {
	p.prompts = prompts
}

// AddPrompt adds the prompt to the list of prompts
func (p *MemoryProvider) AddPrompt(prompt Prompt) {
	if p.prompts == nil {
		p.prompts = Prompts{}
	}

	p.prompts = append(p.prompts, prompt)
}
