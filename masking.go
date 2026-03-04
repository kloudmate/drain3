package drain3

import (
	"time"

	"github.com/dlclark/regexp2"
)

// MaskingInstruction defines a regex pattern and its replacement mask.
type MaskingInstruction struct {
	Pattern    string `yaml:"pattern" json:"pattern"`
	MaskWith   string `yaml:"mask_with" json:"mask_with"`
	MaskPrefix string `yaml:"-" json:"-"`
	MaskSuffix string `yaml:"-" json:"-"`
	regex      *regexp2.Regexp
}

// NewMaskingInstruction creates a new MaskingInstruction with a compiled regex.
func NewMaskingInstruction(pattern, maskWith string) (*MaskingInstruction, error) {
	re, err := regexp2.Compile(pattern, regexp2.RE2)
	if err != nil {
		// Try without RE2 flag for patterns with lookbehinds
		re, err = regexp2.Compile(pattern, regexp2.None)
		if err != nil {
			return nil, err
		}
	}
	return &MaskingInstruction{
		Pattern:  pattern,
		MaskWith: maskWith,
		regex:    re,
	}, nil
}

// LogMasker applies a sequence of masking instructions to log messages.
type LogMasker struct {
	Instructions []*MaskingInstruction
	MaskPrefix   string
	MaskSuffix   string
}

// NewLogMasker creates a new LogMasker with the given prefix and suffix for mask tokens.
func NewLogMasker(instructions []*MaskingInstruction, maskPrefix, maskSuffix string) *LogMasker {
	if maskPrefix == "" {
		maskPrefix = "<"
	}
	if maskSuffix == "" {
		maskSuffix = ">"
	}
	for _, inst := range instructions {
		inst.MaskPrefix = maskPrefix
		inst.MaskSuffix = maskSuffix
	}
	return &LogMasker{
		Instructions: instructions,
		MaskPrefix:   maskPrefix,
		MaskSuffix:   maskSuffix,
	}
}

// Mask applies all masking instructions sequentially to the content.
func (m *LogMasker) Mask(content string) string {
	if m == nil || len(m.Instructions) == 0 {
		return content
	}
	for _, inst := range m.Instructions {
		replacement := inst.MaskPrefix + inst.MaskWith + inst.MaskSuffix
		result, err := inst.regex.Replace(content, replacement, -1, -1)
		if err != nil {
			// On error, skip this instruction
			continue
		}
		content = result
	}
	return content
}

// MaskNames returns all unique mask names from the instructions.
func (m *LogMasker) MaskNames() []string {
	seen := make(map[string]bool)
	var names []string
	for _, inst := range m.Instructions {
		if !seen[inst.MaskWith] {
			seen[inst.MaskWith] = true
			names = append(names, inst.MaskWith)
		}
	}
	return names
}

// InstructionsByMaskName returns all instructions that use the given mask name.
func (m *LogMasker) InstructionsByMaskName(maskName string) []*MaskingInstruction {
	var result []*MaskingInstruction
	for _, inst := range m.Instructions {
		if inst.MaskWith == maskName {
			result = append(result, inst)
		}
	}
	return result
}

// SetTimeout sets a regex match timeout for all masking instructions.
func (m *LogMasker) SetTimeout(timeout time.Duration) {
	for _, inst := range m.Instructions {
		inst.regex.MatchTimeout = timeout
	}
}
