package internal

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/sourcegraph/scip/bindings/go/scip"
	"google.golang.org/protobuf/proto"
)

// SCIPAnalyer Analyze with SCIP index file
//
// Used to load scip index file and analyze source code
// to extract language related file info,
// including hover doc, go to definition info and more.
type SCIPAnalyer struct {
	scipIndex *scip.Index
	symbolMap map[string]*scip.SymbolInformation
}

func NewSCIPAnalyer(indexPath string) (*SCIPAnalyer, error) {
	scipFile, err := os.Open(indexPath)
	if err != nil {
		return nil, err
	}
	defer scipFile.Close()
	scipBytes, err := io.ReadAll(scipFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read SCIP index at path %s", indexPath)
	}

	scipIndex := scip.Index{}
	err = proto.Unmarshal(scipBytes, &scipIndex)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal SCIP index at path %s", indexPath)
	}

	symbolMap := make(map[string]*scip.SymbolInformation)
	for _, doc := range scipIndex.Documents {
		for _, sym := range doc.Symbols {
			symbolMap[sym.Symbol] = sym
		}
	}
	for _, sym := range scipIndex.ExternalSymbols {
		symbolMap[sym.Symbol] = sym
	}

	return &SCIPAnalyer{
		scipIndex: &scipIndex,
		symbolMap: symbolMap,
	}, nil
}

func (s *SCIPAnalyer) Analyze(sourcePath string) []TokenInfo {
	var document *scip.Document

	// Normalize project root from SCIP metadata.
	// If it's a file:// URI, extract the path. Otherwise, treat it as a raw path.
	projectRoot := s.scipIndex.Metadata.ProjectRoot
	if u, err := url.Parse(projectRoot); err == nil && u.Scheme == "file" {
		projectRoot = u.Path
	}

	// Iterate through documents to find a match based on absolute file paths.
	for _, doc := range s.scipIndex.Documents {
		// Construct the absolute path for the document.
		// filepath.FromSlash converts forward slashes (common in SCIP relative paths)
		// to the system-specific path separator (e.g., backslashes on Windows).
		absDocPath := filepath.Join(projectRoot, filepath.FromSlash(doc.RelativePath))

		// Compare with the absolute sourcePath provided to the Analyze method.
		if absDocPath == sourcePath {
			document = doc
			break
		}
	}

	if document == nil {
		return []TokenInfo{}
	}

	var tokens []TokenInfo
	for _, occ := range document.Occurrences {
		span := parseRange(occ.Range)

		isDefinition := (occ.SymbolRoles & int32(scip.SymbolRole_Definition)) != 0
		isReference := !isDefinition

		var inlayText []string
		if symm, ok := s.symbolMap[occ.Symbol]; ok {
			if signatureDoc := symm.SignatureDocumentation; signatureDoc != nil {
				inlayText = append(inlayText, signatureDoc.Text)
			} else {
				if ty := getType(occ.Symbol); ty != "" {
					inlayText = append(inlayText, ty)
				}
			}
		}

		tokens = append(tokens, TokenInfo{
			Symbol:         generateID(occ.Symbol),
			IsReference:    isReference,
			IsDefinition:   isDefinition,
			HighlightClass: "",
			InlayText:      inlayText,
			Span:           span,
		})
	}

	return tokens
}

func getType(symbol string) string {
	typeInfo := ""
	if sym, err := scip.ParseSymbol(symbol); err == nil {
		for _, desc := range sym.Descriptors {
			if desc.Suffix == scip.Descriptor_Type {
				typeInfo = desc.Name
			}
		}
	}
	return typeInfo
}

func parseRange(r []int32) scip.Range {
	if len(r) == 3 {
		return scip.Range{
			Start: scip.Position{Line: r[0], Character: r[1]},
			End:   scip.Position{Line: r[0], Character: r[2]},
		}
	}
	if len(r) == 4 {
		return scip.Range{
			Start: scip.Position{Line: r[0], Character: r[1]},
			End:   scip.Position{Line: r[2], Character: r[3]},
		}
	}
	return scip.Range{}
}

func generateID(symbol string) string {
	var sb strings.Builder
	for _, r := range symbol {
		// Keep safe characters as is
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' || r == '/' || r == '~' {
			sb.WriteRune(r)
		} else if r == ' ' {
			sb.WriteRune('+')
		} else {
			// Percent-encode other characters
			fmt.Fprintf(&sb, "%%%02X", r)
		}
	}
	return sb.String()
}
