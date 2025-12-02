package internal

import (
	"io"
	"net/url"
	"os"

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

	return &SCIPAnalyer{
		scipIndex: &scipIndex,
	}, nil
}

func (s *SCIPAnalyer) Analyze(sourcePath string) []TokenInfo {
	var document *scip.Document
	sourceURI := url.URL{
		Scheme: "file",
		Path:   sourcePath,
	}
	scipRoot := s.scipIndex.Metadata.ProjectRoot
	for _, doc := range s.scipIndex.Documents {
		docRelativePath := doc.RelativePath
		docPath := scipRoot + "/" + docRelativePath
		if docPath == sourceURI.String() {
			document = doc
		}
	}

	for _, symbol := range document.Symbols {
		sym := symbol.Symbol
		symMsg, err := scip.ParseSymbol(sym)
		if err != nil {
			continue
		}

		_ = symMsg.Descriptors // Suppress unused variable warning

		// TODO: Use these variables for proper symbol analysis
		var _ string
		var _ string
		for _, desc := range symMsg.Descriptors {
			if desc.Suffix == scip.Descriptor_Type {
				_ = desc.Name // symType = desc.Name
			}
			_ = desc.Name // displayName = desc.Name
		}
		if symbol.DisplayName != "" {
			_ = symbol.DisplayName // displayName = symbol.DisplayName
		}
	}

	return []TokenInfo{}
}
