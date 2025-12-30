package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AnyAllowlist captures approved any-usage locations for lint enforcement.
type AnyAllowlist struct {
	Version      int                 `json:"version"`
	ExcludeGlobs []string            `json:"exclude_globs"`
	Entries      []AnyAllowlistEntry `json:"entries"`
}

// AnyAllowlistEntry describes a scoped any-usage exception.
type AnyAllowlistEntry struct {
	Path      string   `json:"path"`
	Symbols   []string `json:"symbols,omitempty"`
	Category  string   `json:"category"`
	Public    bool     `json:"public"`
	Rationale string   `json:"rationale"`
	Owner     string   `json:"owner"`
	Refs      []string `json:"refs,omitempty"`
}

var anyAllowlistCategories = map[string]struct{}{
	"json-boundary":      {},
	"third-party-shim":   {},
	"reflection":         {},
	"generic-constraint": {},
	"internal-helper":    {},
	"test-only":          {},
	"legacy-exception":   {},
}

// LoadAnyAllowlist loads an AnyAllowlist from the JSON file at listPath.
// It reads the file, unmarshals its JSON into an AnyAllowlist, validates the
// allowlist structure and contents, and returns the populated allowlist or an
// error describing a read, parse, or validation failure.
func LoadAnyAllowlist(listPath string) (AnyAllowlist, error) {
	// #nosec G304 -- allowlist path is provided by repo tooling during linting
	data, err := os.ReadFile(listPath)
	if err != nil {
		return AnyAllowlist{}, fmt.Errorf("read any allowlist: %w", err)
	}
	var allowlist AnyAllowlist
	if err := json.Unmarshal(data, &allowlist); err != nil {
		return AnyAllowlist{}, fmt.Errorf("parse any allowlist: %w", err)
	}
	if err := validateAllowlist(&allowlist); err != nil {
		return AnyAllowlist{}, err
	}
	return allowlist, nil
}

// ValidateAnyUsageFromFile loads an allowlist from listPath and validates `any` type usages in Go source files under the given roots (relative to baseDir), returning any collected violations or an error.
func ValidateAnyUsageFromFile(listPath, baseDir string, roots []string) ([]Error, error) {
	allowlist, err := LoadAnyAllowlist(listPath)
	if err != nil {
		return nil, err
	}
	return ValidateAnyUsage(allowlist, baseDir, roots)
}

// ValidateAnyUsage reports usages of the `any` type that are not covered by the provided allowlist
// within the given root directories.
//
// It returns a slice of validation errors describing each violation. It returns a non-nil error
// if the allowlist is invalid, the base directory cannot be resolved, a root path is missing or
// not a directory, or filesystem traversal/parsing fails.
func ValidateAnyUsage(allowlist AnyAllowlist, baseDir string, roots []string) ([]Error, error) {
	if len(roots) == 0 {
		return nil, errors.New("no roots provided for any usage validation")
	}
	if err := validateAllowlist(&allowlist); err != nil {
		return nil, err
	}
	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolve base dir: %w", err)
	}
	index := buildAllowlistIndex(allowlist)
	var violations []Error

	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		rootPath := root
		if !filepath.IsAbs(rootPath) {
			rootPath = filepath.Join(baseAbs, rootPath)
		}
		info, err := os.Stat(rootPath)
		if err != nil {
			return nil, fmt.Errorf("stat root %s: %w", root, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("root %s is not a directory", root)
		}
		if err := filepath.WalkDir(rootPath, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			rel, err := filepath.Rel(baseAbs, path)
			if err != nil {
				return err
			}
			rel = normalizePath(rel)
			if shouldExclude(rel, allowlist.ExcludeGlobs) {
				return nil
			}
			if index.allowAll[rel] {
				return nil
			}
			fileViolations, err := validateAnyFile(path, rel, index)
			if err != nil {
				return err
			}
			violations = append(violations, fileViolations...)
			return nil
		}); err != nil {
			return nil, err
		}
	}

	return violations, nil
}

// validateAllowlist validates and normalizes an AnyAllowlist in place.
// 
// It enforces that Version is >= 1 and that each entry contains a non-empty
// Path, a known Category, a non-empty Owner, and a non-empty Rationale.
// If an entry is marked Public, its Category must be either "json-boundary"
// or "legacy-exception". Symbol lists, entry paths, and exclude globs are
// trimmed and normalized according to package normalization rules.
// 
// Returns an error describing the first validation failure encountered.
func validateAllowlist(allowlist *AnyAllowlist) error {
	if allowlist.Version <= 0 {
		return errors.New("any allowlist version must be >= 1")
	}
	for i, entry := range allowlist.Entries {
		entry.Path = strings.TrimSpace(entry.Path)
		if entry.Path == "" {
			return fmt.Errorf("any allowlist entry %d missing path", i)
		}
		entry.Path = normalizePath(entry.Path)
		entry.Category = strings.TrimSpace(entry.Category)
		if entry.Category == "" {
			return fmt.Errorf("any allowlist entry %d missing category", i)
		}
		if _, ok := anyAllowlistCategories[entry.Category]; !ok {
			return fmt.Errorf("any allowlist entry %d has unknown category %q", i, entry.Category)
		}
		entry.Owner = strings.TrimSpace(entry.Owner)
		if entry.Owner == "" {
			return fmt.Errorf("any allowlist entry %d missing owner", i)
		}
		entry.Rationale = strings.TrimSpace(entry.Rationale)
		if entry.Rationale == "" {
			return fmt.Errorf("any allowlist entry %d missing rationale", i)
		}
		if entry.Public && entry.Category != "json-boundary" && entry.Category != "legacy-exception" {
			return fmt.Errorf("any allowlist entry %d public exception must be json-boundary or legacy-exception", i)
		}
		entry.Symbols = normalizeSymbols(entry.Symbols)
		allowlist.Entries[i] = entry
	}
	for i, glob := range allowlist.ExcludeGlobs {
		allowlist.ExcludeGlobs[i] = strings.TrimSpace(glob)
	}
	return nil
}

// normalizePath cleans and normalizes p into a forward-slash-separated path without a leading "./".
// It trims surrounding whitespace, applies filepath.Clean, converts OS path separators to '/', and strips a leading "./" if present.
func normalizePath(p string) string {
	cleaned := filepath.Clean(strings.TrimSpace(p))
	cleaned = filepath.ToSlash(cleaned)
	return strings.TrimPrefix(cleaned, "./")
}

// normalizeSymbols trims whitespace from each symbol, removes empty entries,
// and returns the cleaned slice.
// If the input is empty or all entries are empty after trimming, it returns nil.
func normalizeSymbols(symbols []string) []string {
	if len(symbols) == 0 {
		return nil
	}
	out := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		symbol = strings.TrimSpace(symbol)
		if symbol != "" {
			out = append(out, symbol)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

type anyAllowlistIndex struct {
	allowAll map[string]bool
	symbols  map[string]map[string]struct{}
}

// buildAllowlistIndex constructs an anyAllowlistIndex from the provided AnyAllowlist.
// 
// For each entry, a path with no Symbols is recorded as allowing all usages for that path.
// Entries that specify Symbols populate a per-path set of allowed symbol names.
func buildAllowlistIndex(allowlist AnyAllowlist) anyAllowlistIndex {
	index := anyAllowlistIndex{
		allowAll: make(map[string]bool),
		symbols:  make(map[string]map[string]struct{}),
	}
	for _, entry := range allowlist.Entries {
		if len(entry.Symbols) == 0 {
			index.allowAll[entry.Path] = true
			continue
		}
		symbolSet, ok := index.symbols[entry.Path]
		if !ok {
			symbolSet = make(map[string]struct{})
			index.symbols[entry.Path] = symbolSet
		}
		for _, symbol := range entry.Symbols {
			symbolSet[symbol] = struct{}{}
		}
	}
	return index
}

func (index anyAllowlistIndex) isAllowed(relPath, symbol string) bool {
	if index.allowAll[relPath] {
		return true
	}
	if symbol == "" {
		return false
	}
	symbols, ok := index.symbols[relPath]
	if !ok {
		return false
	}
	_, ok = symbols[symbol]
	return ok
}

type anyUsage struct {
	pos token.Pos
}

// validateAnyFile checks a single Go source file for disallowed uses of the identifier "any"
// according to the provided allowlist index.
 // 
 // The function returns a slice of validation Errors describing each violation found in the
 // file (using relPath for Error.File and the source line for Error.Code), or a non-nil error
 // if the file could not be read or parsed.
 // 
 // Parameters:
 //   - path: filesystem path to the Go source file to inspect.
 //   - relPath: path relative to the repository or base directory used in reported errors.
 //   - index: allowlist index used to determine whether a given "any" usage is permitted.
 //
 // The function does not modify the file system.
func validateAnyFile(path, relPath string, index anyAllowlistIndex) ([]Error, error) {
	// #nosec G304 -- path is derived from repo walk and validated roots
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	typeParamRanges := collectTypeParamRanges(file)
	symbols := buildSymbolRanges(file)
	uses := collectAnyUsages(file, typeParamRanges)
	if len(uses) == 0 {
		return nil, nil
	}
	lines := strings.Split(string(content), "\n")
	var violations []Error
	for _, usage := range uses {
		pos := fset.Position(usage.pos)
		symbol := symbolForPos(symbols, usage.pos)
		if index.isAllowed(relPath, symbol) {
			continue
		}
		code := ""
		if pos.Line > 0 && pos.Line <= len(lines) {
			code = strings.TrimSpace(lines[pos.Line-1])
		}
		violations = append(violations, Error{
			File:    relPath,
			Line:    pos.Line,
			Message: "disallowed any usage; add allowlist entry or replace with a concrete type",
			Code:    code,
		})
	}
	return violations, nil
}

type typeParamRange struct {
	start token.Pos
	end   token.Pos
}

// collectTypeParamRanges returns the token position ranges of type parameter type expressions
// for all function and type declarations found in the provided AST file.
 // The returned slice contains start/end positions covering each type parameter's type expression.
func collectTypeParamRanges(file *ast.File) []typeParamRange {
	var ranges []typeParamRange
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncType:
			ranges = append(ranges, typeParamRanges(node.TypeParams)...)
		case *ast.TypeSpec:
			ranges = append(ranges, typeParamRanges(node.TypeParams)...)
		}
		return true
	})
	return ranges
}

// typeParamRanges returns the start and end token positions for each non-nil
// field's Type in the provided FieldList, representing ranges of type
// parameter type expressions. If fields is nil or contains no valid Type
// entries, it returns nil.
func typeParamRanges(fields *ast.FieldList) []typeParamRange {
	if fields == nil {
		return nil
	}
	var ranges []typeParamRange
	for _, field := range fields.List {
		if field == nil || field.Type == nil {
			continue
		}
		ranges = append(ranges, typeParamRange{
			start: field.Type.Pos(),
			end:   field.Type.End(),
		})
	}
	return ranges
}

// collectAnyUsages returns the positions of identifier usages of "any" that are used as types and not located inside provided type parameter ranges.
// It examines the AST and yields an anyUsage for each "any" identifier that represents a type context and is outside any of the given constraint ranges.
func collectAnyUsages(file *ast.File, constraints []typeParamRange) []anyUsage {
	var uses []anyUsage
	var stack []ast.Node
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		stack = append(stack, n)
		ident, ok := n.(*ast.Ident)
		if ok && ident.Name == "any" && isTypeIdent(stack) && !isInTypeParamRange(ident.Pos(), constraints) {
			uses = append(uses, anyUsage{pos: ident.Pos()})
		}
		return true
	})
	return uses
}

// isInTypeParamRange reports whether pos falls within any of the provided type parameter ranges.
func isInTypeParamRange(pos token.Pos, ranges []typeParamRange) bool {
	for _, r := range ranges {
		if pos >= r.start && pos <= r.end {
			return true
		}
	}
	return false
}

// isTypeIdent reports whether the last node in the provided AST node stack is used in a type position.
// It determines this by examining the immediate parent node and returning true when the child occupies a type context
// such as array element, map key/value, channel element, pointer target, ellipsis element, field/value/type spec,
// type assertion, index/index-list index, or call expression function position.
func isTypeIdent(stack []ast.Node) bool {
	if len(stack) < 2 {
		return false
	}
	parent := stack[len(stack)-2]
	child := stack[len(stack)-1]
	switch node := parent.(type) {
	case *ast.ArrayType:
		return node.Elt == child
	case *ast.MapType:
		return node.Key == child || node.Value == child
	case *ast.ChanType:
		return node.Value == child
	case *ast.StarExpr:
		return node.X == child
	case *ast.Ellipsis:
		return node.Elt == child
	case *ast.Field:
		return node.Type == child
	case *ast.ValueSpec:
		return node.Type == child
	case *ast.TypeSpec:
		return node.Type == child
	case *ast.TypeAssertExpr:
		return node.Type == child
	case *ast.IndexExpr:
		return node.Index == child
	case *ast.IndexListExpr:
		for _, index := range node.Indices {
			if index == child {
				return true
			}
		}
	case *ast.CallExpr:
		return node.Fun == child
	}
	return false
}

type symbolRange struct {
	name  string
	start token.Pos
	end   token.Pos
}

// buildSymbolRanges collects source ranges for declared types, values, and functions in the given AST file.
// The returned slice contains symbolRange entries with the symbol name and its start and end positions for each TypeSpec, each name in a ValueSpec, and each FuncDecl (using the receiver type name for methods when resolvable).
func buildSymbolRanges(file *ast.File) []symbolRange {
	var ranges []symbolRange
	for _, decl := range file.Decls {
		switch node := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					ranges = append(ranges, symbolRange{name: spec.Name.Name, start: spec.Pos(), end: spec.End()})
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						ranges = append(ranges, symbolRange{name: name.Name, start: spec.Pos(), end: spec.End()})
					}
				}
			}
		case *ast.FuncDecl:
			name := node.Name.Name
			if node.Recv != nil && len(node.Recv.List) > 0 {
				if recvName := receiverTypeName(node.Recv.List[0].Type); recvName != "" {
					name = recvName
				}
			}
			ranges = append(ranges, symbolRange{name: name, start: node.Pos(), end: node.End()})
		}
	}
	return ranges
}

// receiverTypeName extracts the base type name from a method receiver expression.
// It supports plain identifiers, pointer receivers, and generic indexed receiver
// forms (e.g., T[K] or T[K1, K2]); returns the type name or an empty string if
// the receiver type cannot be determined.
func receiverTypeName(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.Ident:
		return node.Name
	case *ast.StarExpr:
		return receiverTypeName(node.X)
	case *ast.IndexExpr:
		return receiverTypeName(node.X)
	case *ast.IndexListExpr:
		return receiverTypeName(node.X)
	}
	return ""
}

// symbolForPos returns the name of the symbol whose span includes the given file position.
// It returns an empty string if no symbol range contains pos.
func symbolForPos(ranges []symbolRange, pos token.Pos) string {
	for _, r := range ranges {
		if pos >= r.start && pos <= r.end {
			return r.name
		}
	}
	return ""
}

// shouldExclude reports whether relPath matches any non-empty pattern in globs.
// Empty patterns are ignored; if matchGlob returns an error for a pattern it is treated as no match.
// Returns true as soon as any pattern successfully matches, false otherwise.
func shouldExclude(relPath string, globs []string) bool {
	for _, glob := range globs {
		if glob == "" {
			continue
		}
		matched, err := matchGlob(glob, relPath)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// matchGlob reports whether value matches the glob pattern.
// The pattern and value are normalized to forward-slash paths; the pattern supports
// `*` (matches any sequence of non-separator characters), `?` (matches any single
// non-separator character) and `**` (matches any sequence of characters including
// directory separators). Returns an error if the translated regular expression
// fails to compile.
func matchGlob(pattern, value string) (bool, error) {
	pattern = normalizePath(pattern)
	value = normalizePath(value)
	escaped := regexp.QuoteMeta(pattern)
	escaped = strings.ReplaceAll(escaped, `\*\*`, "<<ANY>>")
	escaped = strings.ReplaceAll(escaped, `\*`, `[^/]*`)
	escaped = strings.ReplaceAll(escaped, `\?`, `[^/]`)
	escaped = strings.ReplaceAll(escaped, "<<ANY>>", ".*")
	expr := "^" + escaped + "$"
	re, err := regexp.Compile(expr)
	if err != nil {
		return false, err
	}
	return re.MatchString(value), nil
}