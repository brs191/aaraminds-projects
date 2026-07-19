package extraction

import (
	"regexp"
	"strings"

	"github.com/aaraminds/dif/libs/sourceanchors"
)

var atxHeadingPattern = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*$`)

// ExtractMarkdown emits deterministic document, section, block, anchor,
// passage, and CONTAINS edge records for a Markdown source file.
func ExtractMarkdown(content string, opts Options) (Result, error) {
	if err := opts.validate(); err != nil {
		return Result{}, err
	}
	if err := ensureNonEmptyContent(content); err != nil {
		return Result{}, err
	}
	sourceLines := lines(content)
	result := newBaseResult(opts, FormatMarkdown, sourceLines)

	documentNode, err := result.addNode(opts, NodeDocument, "", 0, "", 1, len(sourceLines), strings.Join(sourceLines, "\n"), sourceanchors.TypeMarkdown)
	if err != nil {
		return Result{}, err
	}

	headingStack := map[int]string{}
	currentParentID := documentNode.NodeID
	currentHeadingPath := ""
	currentHeadingLine := 0
	ordinal := 1
	index := 0
	for index < len(sourceLines) {
		lineNumber := index + 1
		line := sourceLines[index]
		if level, title, ok := parseHeading(line); ok {
			for existingLevel := range headingStack {
				if existingLevel >= level {
					delete(headingStack, existingLevel)
				}
			}
			headingStack[level] = title
			currentHeadingPath = joinHeadingPath(headingStack)
			sectionNode, err := result.addNode(opts, NodeSection, documentNode.NodeID, ordinal, currentHeadingPath, lineNumber, lineNumber, line, sourceanchors.TypeMarkdown)
			if err != nil {
				return Result{}, err
			}
			ordinal++
			currentParentID = sectionNode.NodeID
			currentHeadingLine = lineNumber
			index++
			continue
		}
		if strings.TrimSpace(line) == "" {
			index++
			continue
		}
		start := lineNumber
		for index < len(sourceLines) {
			if strings.TrimSpace(sourceLines[index]) == "" {
				break
			}
			if _, _, ok := parseHeading(sourceLines[index]); ok {
				break
			}
			index++
		}
		end := index
		anchorStart := start
		anchorEnd := end
		if currentHeadingLine > 0 {
			anchorStart = currentHeadingLine
			if anchorEnd > start {
				anchorEnd--
			}
		}
		text := lineRangeText(sourceLines, anchorStart, anchorEnd)
		if _, err := result.addNode(opts, NodeBlock, currentParentID, ordinal, currentHeadingPath, anchorStart, anchorEnd, text, sourceanchors.TypeMarkdown); err != nil {
			return Result{}, err
		}
		ordinal++
	}
	return result, nil
}

func parseHeading(line string) (int, string, bool) {
	match := atxHeadingPattern.FindStringSubmatch(line)
	if match == nil {
		return 0, "", false
	}
	title := strings.TrimSpace(strings.TrimRight(match[2], "#"))
	if title == "" {
		return 0, "", false
	}
	return len(match[1]), title, true
}

func joinHeadingPath(stack map[int]string) string {
	parts := make([]string, 0, len(stack))
	for level := 1; level <= 6; level++ {
		if value := strings.TrimSpace(stack[level]); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, " > ")
}
