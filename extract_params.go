package drain3

import (
	"regexp"
	"strings"
	"sync"
)

// ExtractedParameter represents a single extracted parameter value and its mask name.
type ExtractedParameter struct {
	Value    string
	MaskName string
}

// ParameterExtractor extracts variable values from log messages given a template.
type ParameterExtractor struct {
	masker          *LogMasker
	extraDelimiters []string
	cache           sync.Map // map[string]*extractionRegex
}

type extractionRegex struct {
	regex           *regexp.Regexp
	groupToMaskName map[string]string
}

// NewParameterExtractor creates a new ParameterExtractor.
func NewParameterExtractor(masker *LogMasker, extraDelimiters []string) *ParameterExtractor {
	return &ParameterExtractor{
		masker:          masker,
		extraDelimiters: extraDelimiters,
	}
}

// ExtractParameters extracts parameter values from a log message according to a template.
// If exactMatching is true and a masker is provided, mask-specific patterns are used.
// Returns nil if the message does not match the template.
func (pe *ParameterExtractor) ExtractParameters(logTemplate, logMessage string, exactMatching bool) []ExtractedParameter {
	for _, delim := range pe.extraDelimiters {
		logMessage = strings.ReplaceAll(logMessage, delim, " ")
	}

	cacheKey := logTemplate
	if exactMatching {
		cacheKey += ":exact"
	}

	// Check cache
	var er *extractionRegex
	if cached, ok := pe.cache.Load(cacheKey); ok {
		er = cached.(*extractionRegex)
	} else {
		er = pe.buildExtractionRegex(logTemplate, exactMatching)
		pe.cache.Store(cacheKey, er)
	}

	matches := er.regex.FindStringSubmatch(logMessage)
	if matches == nil {
		return nil
	}

	var params []ExtractedParameter
	for i, name := range er.regex.SubexpNames() {
		if name == "" || i >= len(matches) {
			continue
		}
		maskName, ok := er.groupToMaskName[name]
		if !ok {
			continue
		}
		params = append(params, ExtractedParameter{
			Value:    matches[i],
			MaskName: maskName,
		})
	}

	return params
}

func (pe *ParameterExtractor) buildExtractionRegex(logTemplate string, exactMatching bool) *extractionRegex {
	groupToMaskName := make(map[string]string)
	paramCounter := 0

	// Escape the template for regex
	escaped := regexp.QuoteMeta(logTemplate)

	// Replace whitespace with flexible whitespace matcher
	escaped = strings.ReplaceAll(escaped, "\\ ", `\s+`)

	// Collect all mask names we need to handle
	maskNames := make(map[string]bool)
	maskNames["*"] = true // The Drain catch-all
	if pe.masker != nil {
		for _, name := range pe.masker.MaskNames() {
			maskNames[name] = true
		}
	}

	prefix := "<"
	suffix := ">"
	if pe.masker != nil {
		prefix = pe.masker.MaskPrefix
		suffix = pe.masker.MaskSuffix
	}

	// Replace each mask placeholder with a named capture group
	for maskName := range maskNames {
		searchStr := regexp.QuoteMeta(prefix + maskName + suffix)
		for strings.Contains(escaped, searchStr) {
			groupName := paramGroupName(paramCounter)
			paramCounter++
			groupToMaskName[groupName] = maskName

			var capturePattern string
			if exactMatching && maskName != "*" && pe.masker != nil {
				// Use mask-specific patterns
				capturePattern = ".+?"
			} else {
				capturePattern = ".+?"
			}

			replacement := "(?P<" + groupName + ">" + capturePattern + ")"
			escaped = strings.Replace(escaped, searchStr, replacement, 1)
		}
	}

	escaped = "^" + escaped + "$"
	re, err := regexp.Compile(escaped)
	if err != nil {
		// Fallback: compile a non-matching regex
		re = regexp.MustCompile("^$")
	}

	return &extractionRegex{
		regex:           re,
		groupToMaskName: groupToMaskName,
	}
}

func paramGroupName(index int) string {
	return "p" + intToStr(index)
}
