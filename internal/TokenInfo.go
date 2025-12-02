package internal

import (
	"sort"

	"github.com/cockroachdb/errors"
	"github.com/sourcegraph/scip/bindings/go/scip"
)

// TokenInfo represents information about a symbol in code, including its position and attributes
type TokenInfo struct {
	Symbol         string     // Symbol name or identifier
	IsReference    bool       // Whether this is a reference
	IsDefinition   bool       // Whether this is a definition
	HighlightClass string     // Syntax highlighting class
	InlayText      []string   // Inlay Text, for example, type info
	Span           scip.Range // Position range of the symbol in code
}

// SortTokens sorts tokens primarily by start position, then by end position in reverse order
func SortTokens(tokens []TokenInfo) {
	sort.Slice(tokens, func(i, j int) bool {
		s := scip.Position.Compare(tokens[i].Span.Start, tokens[j].Span.Start)
		if s != 0 {
			return s < 0
		} else {
			return scip.Position.Less(tokens[i].Span.End, tokens[j].Span.End)
		}
	})
}

// MergeSplitTokens merges overlapping tokens and splits them at intersection points to eliminate overlaps
func MergeSplitTokens(tokens []TokenInfo) ([]TokenInfo, error) {
	if len(tokens) == 0 {
		return []TokenInfo{}, nil
	}
	result := []TokenInfo{}
	activeTokens := []TokenInfo{}
	curPos := tokens[0].Span.Start
	index := 0
	for {
		if index >= len(tokens) && len(activeTokens) == 0 {
			break
		}

		nextSplit, err := findNextSplit(tokens, index, activeTokens)
		if err != nil {
			return nil, err
		}

		if scip.Position.Compare(curPos, nextSplit) < 0 {
			segment := createSegment(curPos, nextSplit, activeTokens)
			if segment != nil {
				result = append(result, *segment)
			}
		}

		newIndex, newActiveTokens, err := processSplitAtPosition(nextSplit, tokens, index, activeTokens)
		if err != nil {
			return nil, err
		}

		activeTokens = newActiveTokens
		curPos = nextSplit
		index = newIndex
	}
	return result, nil
}

// processSplitAtPosition processes tokens that start or end at the given position and updates the active tokens list
func processSplitAtPosition(pos scip.Position, tokens []TokenInfo, index int, activeTokens []TokenInfo) (int, []TokenInfo, error) {
	newSplitIndex := index
	newActiveTokens := activeTokens

	for {
		if newSplitIndex >= len(tokens) || scip.Position.Compare(tokens[newSplitIndex].Span.Start, pos) != 0 {
			break
		}
		newActiveTokens = append(newActiveTokens, tokens[newSplitIndex])
		newSplitIndex++
	}

	n := 0
	for _, token := range newActiveTokens {
		if scip.Position.Compare(token.Span.End, pos) != 0 {
			newActiveTokens[n] = token
			n++
		}
	}
	newActiveTokens = newActiveTokens[:n]

	return newSplitIndex, newActiveTokens, nil
}

// findNextSplit finds the next token start position or the earliest token end position
func findNextSplit(tokens []TokenInfo, index int, activeTokens []TokenInfo) (scip.Position, error) {
	var nextPos scip.Position
	isEndEvent := false

	if index < len(tokens) {
		nextPos = tokens[index].Span.Start
	} else if len(activeTokens) > 0 {
		nextPos = findNextEnd(activeTokens)
		isEndEvent = true
	} else {
		return scip.Position{}, errors.Errorf("Should not hit in findNextSplit")
	}

	if len(activeTokens) > 0 && !isEndEvent {
		nextEnd := findNextEnd(activeTokens)
		if scip.Position.Compare(nextEnd, nextPos) < 0 {
			return nextEnd, nil
		}
	}

	return nextPos, nil
}

// findNextEnd finds the earliest ending position among all active tokens
func findNextEnd(activeTokens []TokenInfo) scip.Position {
	earliest := activeTokens[0].Span.End
	for _, token := range activeTokens {
		if scip.Position.Compare(token.Span.End, earliest) < 0 {
			earliest = token.Span.End
		}
	}
	return earliest
}

// createSegment creates a new TokenInfo segment by merging properties from active tokens within the given range
func createSegment(start scip.Position, end scip.Position, activeTokens []TokenInfo) *TokenInfo {
	var result TokenInfo
	if len(activeTokens) == 0 {
		return nil
	}

	for _, token := range activeTokens {
		if token.Symbol != "" {
			result.Symbol = token.Symbol
		}
		if token.HighlightClass != "" {
			result.HighlightClass = token.HighlightClass
		}
		if len(token.InlayText) > 0 {
			result.InlayText = append(result.InlayText, token.InlayText...)
		}
		result.IsReference = result.IsReference || token.IsReference
		result.IsDefinition = result.IsDefinition || token.IsDefinition
	}

	result.Span = scip.Range{Start: start, End: end}
	return &result
}
