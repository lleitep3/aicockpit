package tracking

import (
	"bufio"
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
	// Build header string.
	header := fmt.Sprintf("// package:%s version:%s created:%s updated:%s", pkg.Name, pkg.Version, pkg.Metadata.CreationDate, pkg.Metadata.LastModified)

	// Read existing content.
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("tracking: cannot open file %s: %w", filePath, err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("tracking: error reading file %s: %w", filePath, err)
	}

	// Prepend header.
	out, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("tracking: cannot open file for writing %s: %w", filePath, err)
	}
	defer out.Close()
	writer := bufio.NewWriter(out)
	if _, err := writer.WriteString(header + "\n"); err != nil {
		return fmt.Errorf("tracking: cannot write header to %s: %w", filePath, err)
	}
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("tracking: cannot write content to %s: %w", filePath, err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("tracking: flush error for %s: %w", filePath, err)
	}
	return nil
}
