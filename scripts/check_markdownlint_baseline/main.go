// Package markdownlintbaseline compares markdownlint JSON output against a baseline
// while lint unification (#116) and golangci stdlib alignment (#114) are in flight.
package markdownlintbaseline

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

const (
	defaultTool    = "markdownlint-cli"
	defaultConfig  = ".markdownlint.yaml"
	defaultScope   = "**/*.md"
	defaultUsage   = "Usage: check_markdownlint_baseline --baseline <file> [--input <file>] [--update]\n"
	defaultRunHint = "Update the baseline with: make lint-docs-update\n"
)

type lintIssue struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Rule string `json:"rule"`
}

type baselineFile struct {
	Meta   baselineMeta `json:"meta"`
	Issues []lintIssue  `json:"issues"`
}

type baselineMeta struct {
	Tool             string                    `json:"tool,omitempty"`
	Config           string                    `json:"config,omitempty"`
	Scope            string                    `json:"scope,omitempty"`
	References       []string                  `json:"references,omitempty"`
	Notes            string                    `json:"notes,omitempty"`
	Owner            string                    `json:"baseline_owner,omitempty"`
	ReductionTargets []baselineReductionTarget `json:"reduction_targets,omitempty"`
	NextReviewDate   string                    `json:"next_review_date,omitempty"`
	Workflow         string                    `json:"workflow,omitempty"`
	CIPolicy         string                    `json:"ci_policy,omitempty"`
}

type baselineReductionTarget struct {
	Milestone     string `json:"milestone"`
	TargetPercent int    `json:"target_percent"`
	DueDate       string `json:"due_date"`
}

type markdownlintMessage struct {
	LineNumber int      `json:"lineNumber"`
	RuleNames  []string `json:"ruleNames"`
}

type markdownlintFlatMessage struct {
	FileName   string   `json:"fileName"`
	LineNumber int      `json:"lineNumber"`
	RuleNames  []string `json:"ruleNames"`
}

// Run executes the markdownlint baseline check with the provided args and IO.
func Run(args []string, stderr io.Writer, stdin io.Reader) int {
	fs := flag.NewFlagSet("check_markdownlint_baseline", flag.ContinueOnError)
	fs.SetOutput(stderr)
	baselinePath := fs.String("baseline", "", "path to markdownlint baseline file")
	inputPath := fs.String("input", "", "path to markdownlint JSON output (defaults to stdin)")
	update := fs.Bool("update", false, "overwrite the baseline with the current lint results")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}

	if strings.TrimSpace(*baselinePath) == "" {
		if _, err := io.WriteString(stderr, defaultUsage); err != nil {
			return 1
		}
		return 2
	}

	data, err := readLintInput(*inputPath, stdin)
	if err != nil {
		return reportError(stderr, "read lint output: %v\n", err)
	}

	issues, err := parseMarkdownlintOutput(data)
	if err != nil {
		return reportError(stderr, "parse lint output: %v\n", err)
	}
	issues = normalizeIssues(issues)

	if *update {
		meta := defaultBaselineMeta()
		if existing, err := loadBaseline(*baselinePath); err == nil {
			meta = mergeMeta(existing.Meta, meta)
		}
		if err := writeBaseline(*baselinePath, meta, issues); err != nil {
			return reportError(stderr, "write baseline: %v\n", err)
		}
		return 0
	}

	baseline, err := loadBaseline(*baselinePath)
	if err != nil {
		return reportError(stderr, "load baseline: %v\n", err)
	}

	newIssues := diffIssues(issues, normalizeIssues(baseline.Issues))
	if len(newIssues) > 0 {
		var builder strings.Builder
		fmt.Fprintf(&builder, "Found %d new markdownlint issue(s):\n", len(newIssues))
		for _, issue := range newIssues {
			fmt.Fprintf(&builder, "- %s:%d %s\n", issue.File, issue.Line, issue.Rule)
		}
		builder.WriteString(defaultRunHint)
		return reportError(stderr, "%s", builder.String())
	}
	return 0
}

func readLintInput(path string, stdin io.Reader) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return io.ReadAll(stdin)
	}
	// #nosec G304 -- path comes from the --input flag; this CLI tool accepts user-supplied paths.
	return os.ReadFile(path)
}

func parseMarkdownlintOutput(data []byte) ([]lintIssue, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}
	var raw map[string][]markdownlintMessage
	rawErr := json.Unmarshal(data, &raw)
	if rawErr == nil {
		if len(raw) == 0 {
			return nil, nil
		}
		total := 0
		for _, messages := range raw {
			total += len(messages)
		}
		issues := make([]lintIssue, 0, total)
		for file, messages := range raw {
			for _, msg := range messages {
				rule := firstRuleName(msg.RuleNames)
				if rule == "" {
					return nil, fmt.Errorf("missing rule name for %s:%d", file, msg.LineNumber)
				}
				issues = append(issues, lintIssue{
					File: file,
					Line: msg.LineNumber,
					Rule: rule,
				})
			}
		}
		return issues, nil
	}

	var flat []markdownlintFlatMessage
	flatErr := json.Unmarshal(data, &flat)
	if flatErr != nil {
		return nil, fmt.Errorf("parse markdownlint output: map error: %v; list error: %v", rawErr, flatErr)
	}
	issues := make([]lintIssue, 0, len(flat))
	for _, msg := range flat {
		rule := firstRuleName(msg.RuleNames)
		if rule == "" {
			return nil, fmt.Errorf("missing rule name for %s:%d", msg.FileName, msg.LineNumber)
		}
		issues = append(issues, lintIssue{
			File: msg.FileName,
			Line: msg.LineNumber,
			Rule: rule,
		})
	}
	return issues, nil
}

func firstRuleName(ruleNames []string) string {
	for _, rule := range ruleNames {
		if strings.TrimSpace(rule) != "" {
			return rule
		}
	}
	return ""
}

func normalizeIssues(issues []lintIssue) []lintIssue {
	seen := make(map[lintIssue]struct{}, len(issues))
	normalized := make([]lintIssue, 0, len(issues))
	for _, issue := range issues {
		if strings.TrimSpace(issue.File) == "" || issue.Line == 0 || strings.TrimSpace(issue.Rule) == "" {
			continue
		}
		if _, ok := seen[issue]; ok {
			continue
		}
		seen[issue] = struct{}{}
		normalized = append(normalized, issue)
	}
	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].File != normalized[j].File {
			return normalized[i].File < normalized[j].File
		}
		if normalized[i].Line != normalized[j].Line {
			return normalized[i].Line < normalized[j].Line
		}
		return normalized[i].Rule < normalized[j].Rule
	})
	return normalized
}

func loadBaseline(path string) (baselineFile, error) {
	// #nosec G304 -- baseline path comes from the --baseline flag; this CLI tool accepts user-supplied paths.
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return baselineFile{}, fmt.Errorf("baseline not found (%s); run lint-docs-update", path)
		}
		return baselineFile{}, err
	}

	var baseline baselineFile
	if err := json.Unmarshal(data, &baseline); err == nil {
		return baseline, nil
	}

	var issues []lintIssue
	if err := json.Unmarshal(data, &issues); err != nil {
		return baselineFile{}, err
	}
	return baselineFile{Issues: issues}, nil
}

func writeBaseline(path string, meta baselineMeta, issues []lintIssue) error {
	payload := baselineFile{
		Meta:   meta,
		Issues: normalizeIssues(issues),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	// #nosec G306 -- baseline is a non-sensitive CI artifact that should be readable.
	return os.WriteFile(path, data, 0o644)
}

func diffIssues(current, baseline []lintIssue) []lintIssue {
	if len(current) == 0 {
		return nil
	}
	baselineSet := make(map[lintIssue]struct{}, len(baseline))
	for _, issue := range baseline {
		baselineSet[issue] = struct{}{}
	}
	var delta []lintIssue
	for _, issue := range current {
		if _, ok := baselineSet[issue]; !ok {
			delta = append(delta, issue)
		}
	}
	return normalizeIssues(delta)
}

func defaultBaselineMeta() baselineMeta {
	return baselineMeta{
		Tool:       defaultTool,
		Config:     defaultConfig,
		Scope:      defaultScope,
		References: defaultReferences(),
		Notes:      "Temporary docs lint baseline; see #116 for consolidation.",
		Owner:      "Core Maintainers",
		ReductionTargets: []baselineReductionTarget{
			{Milestone: "2026-02", TargetPercent: 25, DueDate: "2026-02-01"},
			{Milestone: "2026-03", TargetPercent: 50, DueDate: "2026-03-01"},
		},
		NextReviewDate: "2026-02-01",
		Workflow:       "docs/annex/0005-documentation-standards.md (#116)",
		CIPolicy:       "manual",
	}
}

func isZeroMeta(meta baselineMeta) bool {
	return meta.Tool == "" &&
		meta.Config == "" &&
		meta.Scope == "" &&
		len(meta.References) == 0 &&
		meta.Notes == "" &&
		meta.Owner == "" &&
		len(meta.ReductionTargets) == 0 &&
		meta.NextReviewDate == "" &&
		meta.Workflow == "" &&
		meta.CIPolicy == ""
}

func reportError(w io.Writer, format string, args ...any) int {
	if _, err := fmt.Fprintf(w, format, args...); err != nil {
		return 1
	}
	return 1
}

func defaultReferences() []string {
	return []string{"#114", "#116"}
}

func mergeMeta(existing, defaults baselineMeta) baselineMeta {
	merged := defaults
	if existing.Tool != "" {
		merged.Tool = existing.Tool
	}
	if existing.Config != "" {
		merged.Config = existing.Config
	}
	if existing.Scope != "" {
		merged.Scope = existing.Scope
	}
	if len(existing.References) > 0 {
		merged.References = existing.References
	}
	if existing.Notes != "" {
		merged.Notes = existing.Notes
	}
	if existing.Owner != "" {
		merged.Owner = existing.Owner
	}
	if len(existing.ReductionTargets) > 0 {
		merged.ReductionTargets = existing.ReductionTargets
	}
	if existing.NextReviewDate != "" {
		merged.NextReviewDate = existing.NextReviewDate
	}
	if existing.Workflow != "" {
		merged.Workflow = existing.Workflow
	}
	if existing.CIPolicy != "" {
		merged.CIPolicy = existing.CIPolicy
	}
	return merged
}
