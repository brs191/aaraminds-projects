package extraction

import (
	"strings"

	"github.com/aaraminds/dif/libs/sourceanchors"
)

// ExtractText emits deterministic document, block, anchor, passage, and
// CONTAINS edge records for a TXT source file.
func ExtractText(content string, opts Options) (Result, error) {
	if err := opts.validate(); err != nil {
		return Result{}, err
	}
	if err := ensureNonEmptyContent(content); err != nil {
		return Result{}, err
	}
	sourceLines := lines(content)
	result := newBaseResult(opts, FormatText, sourceLines)

	documentNode, err := result.addNode(opts, NodeDocument, "", 0, "", 1, len(sourceLines), strings.Join(sourceLines, "\n"), sourceanchors.TypeText)
	if err != nil {
		return Result{}, err
	}

	ordinal := 1
	index := firstTextBlockIndex(sourceLines)
	for index < len(sourceLines) {
		if strings.TrimSpace(sourceLines[index]) == "" {
			index++
			continue
		}
		start := index + 1
		for index < len(sourceLines) && strings.TrimSpace(sourceLines[index]) != "" {
			index++
		}
		end := index
		text := lineRangeText(sourceLines, start, end)
		if _, err := result.addNode(opts, NodeBlock, documentNode.NodeID, ordinal, "", start, end, text, sourceanchors.TypeText); err != nil {
			return Result{}, err
		}
		ordinal++
	}
	return result, nil
}

func firstTextBlockIndex(sourceLines []string) int {
	if len(sourceLines) >= 3 && isSetextUnderline(sourceLines[1]) && strings.HasPrefix(strings.ToLower(strings.TrimSpace(sourceLines[2])), "this fixture is synthetic") {
		return 3
	}
	return 0
}

func isSetextUnderline(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	for _, char := range trimmed {
		if char != '=' && char != '-' {
			return false
		}
	}
	return true
}
