package tracking

import (
	"fmt"
	"os"

	"github.com/lleitep3/aicockpit/internal/packages"
)

// GenerateHeader creates the header comment string for a package.
func GenerateHeader(pkg *packages.Package) string {
	return fmt.Sprintf("// package:%s version:%s created:%s updated:%s", pkg.Name, pkg.Version, pkg.Metadata.CreationDate, pkg.Metadata.LastModified)
}

// InjectHeader prepends a header comment to the given file describing the package metadata.
// The header format is:
//
//	// package:<name> version:<version> created:<creation_date> updated:<last_modified>
func InjectHeader(filePath string, pkg *packages.Package) error {
	header := GenerateHeader(pkg) + "\n"

	existing, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("tracking: cannot read file %s: %w", filePath, err)
	}

	content := append([]byte(header), existing...)
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		return fmt.Errorf("tracking: cannot write file %s: %w", filePath, err)
	}

	return nil
}
