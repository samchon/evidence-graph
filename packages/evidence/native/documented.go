package evidence

import (
	"encoding/json"
	"sort"
	"strings"

	shimast "github.com/microsoft/typescript-go/shim/ast"

	"github.com/samchon/ttsc/packages/lint/rule"
)

// documentedRule requires a JSDoc block on every selected export.
//
// Evidence declarations are read from one place only — the JSDoc blocks a node
// reports — so an export without a block is not merely undocumented, it is
// structurally unable to carry a citation. The graph never says so: coverage is
// counted from the evidence side, so an obligation is discharged by whichever
// host does have a block while every undocumented export in the same claim
// contributes nothing and looks exactly like a passing file.
type documentedRule struct{}

type documentedOptions struct {
	Symbol json.RawMessage `json:"symbol"`
}

func (documentedRule) Name() string { return documentedRuleName }

func (documentedRule) Visits() []shimast.Kind {
	return []shimast.Kind{shimast.KindSourceFile}
}

func (documentedRule) NeedsTypeChecker() bool { return false }

func (documentedRule) VisitsDeclarationFiles() bool { return false }

func (documentedRule) Check(ctx *rule.Context, node *shimast.Node) {
	if ctx == nil || ctx.File == nil || node == nil {
		return
	}
	if node.Kind != shimast.KindSourceFile {
		return
	}
	selected, problems := decodeDocumentedOptions(ctx.Options)
	if len(problems) != 0 {
		for _, problem := range problems {
			ctx.Report(node, problem)
		}
		return
	}
	for _, host := range documentedHosts(ctx.File) {
		if !host.Symbols.intersects(selected) {
			continue
		}
		state, block := host.jsdocState(ctx.File)
		switch state {
		case jsdocPresent:
			continue
		case jsdocEmpty:
			ctx.Report(
				block,
				"Empty JSDoc on "+host.describe()+
					". The block states nothing and carries no tag, so it documents nothing and cites nothing. Describe what this declaration is for, or cite its evidence with '@evidence <target> <reason>'.",
			)
		default:
			ctx.Report(
				host.Node,
				"Missing JSDoc on "+host.describe()+
					". An '@evidence' tag is only ever read from a JSDoc block, so without one this declaration can never cite anything. Add a '/** ... */' block above it.",
			)
		}
	}
}

func init() { rule.Register(documentedRule{}) }

func decodeDocumentedOptions(raw json.RawMessage) (symbolSet, []string) {
	options := documentedOptions{}
	if len(strings.TrimSpace(string(raw))) != 0 {
		if err := json.Unmarshal(raw, &options); err != nil {
			return nil, []string{
				"Invalid evidence/documented configuration: expected an IEvidenceDocumentedConfig object, or a bare severity for the default selection.",
			}
		}
		if problems := rejectUnknownDocumentedFields(raw); len(problems) != 0 {
			return nil, problems
		}
	}
	selected, problems := decodeSymbols(
		options.Symbol,
		artifactTypeScript,
		false,
		"symbol",
	)
	if len(problems) != 0 {
		return nil, problems
	}
	return selected, nil
}

func rejectUnknownDocumentedFields(raw json.RawMessage) []string {
	object, problem := decodeObject(raw, "configuration")
	if problem != "" {
		return []string{
			"Invalid evidence/documented configuration: expected an IEvidenceDocumentedConfig object, or a bare severity for the default selection.",
		}
	}
	return rejectUnknownFields(object, []string{"symbol"}, "configuration")
}

// documentedHost is one public declaration that must carry a JSDoc block.
//
// Nodes is plural because an overload set is one declaration to a reader and
// several nodes to the parser. Documenting any one of its signatures documents
// the callable, so the whole run is judged together.
type documentedHost struct {
	Node    *shimast.Node
	Nodes   []*shimast.Node
	Symbols symbolSet
	Names   []string
}

// jsdocState answers for the whole host, accepting a block on any of its nodes.
func (host documentedHost) jsdocState(
	file *shimast.SourceFile,
) (jsdocPresence, *shimast.Node) {
	state := jsdocMissing
	var empty *shimast.Node
	for _, node := range host.Nodes {
		presence, block := jsdocState(file, node)
		if presence == jsdocPresent {
			return jsdocPresent, block
		}
		if presence == jsdocEmpty && empty == nil {
			state = jsdocEmpty
			empty = block
		}
	}
	return state, empty
}

func (host documentedHost) describe() string {
	kind := host.Symbols.names()
	if len(host.Names) == 0 {
		return "exported " + kind
	}
	return "exported " + kind + " '" + strings.Join(host.Names, "', '") + "'"
}

// documentedHosts lists the public declarations of a file in source order.
//
// Classification is delegated to the same collector `evidence/graph` uses, so
// the population that must be able to hold a tag cannot drift from the
// population a claim can select as a host. Reimplementing the walk here would
// let the two disagree, and the rule would then guarantee the wrong set.
func documentedHosts(file *shimast.SourceFile) []documentedHost {
	supported := map[*shimast.Node]symbolSet{}
	collectTypeScriptStatements(
		file.Statements,
		nil,
		"",
		&artifactInventory{Type: artifactTypeScript},
		supported,
		map[string]*evidenceUnit{},
		file.IsDeclarationFile,
		false,
		false,
	)
	hosts := make([]documentedHost, 0, len(supported))
	for node, symbols := range supported {
		if node == nil || len(symbols) == 0 {
			continue
		}
		// A variable's leading JSDoc attaches to the statement wrapper, which is
		// registered as a host beside each of its declarations. Reporting the
		// declarations too would demand a block in a position TypeScript never
		// reads one from.
		if node.Kind == shimast.KindVariableDeclaration {
			continue
		}
		hosts = append(hosts, documentedHost{
			Node:    node,
			Nodes:   []*shimast.Node{node},
			Symbols: symbols,
			Names:   hostDeclaredNames(node),
		})
	}
	sort.Slice(hosts, func(left int, right int) bool {
		return hosts[left].Node.Pos() < hosts[right].Node.Pos()
	})
	return mergeOverloadHosts(hosts)
}

// mergeOverloadHosts folds an overload set into the single host it reads as.
//
// TypeScript requires a callable's overload signatures to be adjacent to their
// implementation, so a run of neighbouring function declarations sharing a name
// is exactly one overload set. Judging the signatures separately would report
// every properly documented overload set in existence, since the convention is
// one block above the first signature.
func mergeOverloadHosts(hosts []documentedHost) []documentedHost {
	merged := make([]documentedHost, 0, len(hosts))
	for _, host := range hosts {
		previous := len(merged) - 1
		if previous >= 0 &&
			host.Node.Kind == shimast.KindFunctionDeclaration &&
			merged[previous].Node.Kind == shimast.KindFunctionDeclaration &&
			len(host.Names) == 1 &&
			len(merged[previous].Names) == 1 &&
			host.Names[0] == merged[previous].Names[0] {
			merged[previous].Nodes = append(merged[previous].Nodes, host.Node)
			continue
		}
		merged = append(merged, host)
	}
	return merged
}

func hostDeclaredNames(node *shimast.Node) []string {
	if node.Kind == shimast.KindVariableStatement {
		names := []string{}
		for _, declared := range topLevelDeclaredNames(node) {
			names = append(names, declared.Name)
		}
		return names
	}
	if name := declarationName(node.Name()); name != "" {
		return []string{name}
	}
	return nil
}

type jsdocPresence int

const (
	jsdocMissing jsdocPresence = iota
	jsdocEmpty
	jsdocPresent
)

// jsdocState reports whether a declaration carries a JSDoc block with content.
//
// Presence is read through the same accessor the tag collector uses, so the
// rule accepts exactly what a citation could be found in. Accepting anything
// wider would pass a comment the graph cannot read, which is the silence this
// rule exists to remove.
func jsdocState(
	file *shimast.SourceFile,
	node *shimast.Node,
) (jsdocPresence, *shimast.Node) {
	content := file.Text()
	state := jsdocMissing
	var empty *shimast.Node
	for _, doc := range node.JSDoc(file) {
		if doc == nil || doc.Pos() < 0 || doc.End() > len(content) {
			continue
		}
		if jsdocHasContent(content[doc.Pos():doc.End()]) {
			return jsdocPresent, doc
		}
		state = jsdocEmpty
		empty = doc
	}
	return state, empty
}

func jsdocHasContent(comment string) bool {
	comment = strings.TrimSpace(comment)
	comment = strings.TrimPrefix(comment, "/**")
	comment = strings.TrimPrefix(comment, "/*")
	comment = strings.TrimSuffix(comment, "*/")
	for _, line := range strings.Split(comment, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if line != "" {
			return true
		}
	}
	return false
}
