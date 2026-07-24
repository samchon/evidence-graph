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
		if !selected.contains(host.Symbol) {
			continue
		}
		switch host.jsdocState(ctx.File) {
		case jsdocPresent:
			continue
		case jsdocEmpty:
			// Anchored on the declaration rather than on the block, like every
			// other finding here. `Report` skips a node's leading trivia, and a
			// JSDoc node is entirely trivia to the declaration it precedes, so
			// anchoring there asks the host to underline a range it is built to
			// skip past.
			ctx.Report(
				host.Node,
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

// documentedHost is one public identity that must carry a JSDoc block.
//
// The unit judged is an identity, never a declaration node. Declaration merging
// and overload sets give one identity several nodes — `interface I` beside
// `namespace I`, a callable beside its signatures — and a block on any of them
// documents the identity. Judging the nodes instead would demand a second block
// on the namespace half of the very idiom `evidence/singular` blesses.
type documentedHost struct {
	Node    *shimast.Node
	Nodes   []*shimast.Node
	Symbol  string
	Targets []string
}

// jsdocState answers for the whole host, accepting a block on any of its nodes.
func (host documentedHost) jsdocState(file *shimast.SourceFile) jsdocPresence {
	state := jsdocMissing
	for _, node := range host.Nodes {
		switch jsdocState(file, node) {
		case jsdocPresent:
			return jsdocPresent
		case jsdocEmpty:
			state = jsdocEmpty
		}
	}
	return state
}

func (host documentedHost) describe() string {
	return "exported " + host.Symbol + " '" +
		strings.Join(host.Targets, "', '") + "'"
}

// documentedHosts lists the public identities of a file in source order.
//
// Both the population and its qualified names come from the collector
// `evidence/graph` uses. The population that must be able to hold a tag is by
// definition the population a claim can select as a host, and the name a
// diagnostic asks the author to cite has to be the name the graph resolves — a
// second walk here would let either drift, and the rule would then guarantee
// the wrong set under the wrong names.
func documentedHosts(file *shimast.SourceFile) []documentedHost {
	inventory := &artifactInventory{
		Type:      artifactTypeScript,
		UnitNodes: map[string][]*shimast.Node{},
	}
	collectTypeScriptStatements(
		file.Statements,
		nil,
		"",
		inventory,
		map[*shimast.Node]symbolSet{},
		map[string]*evidenceUnit{},
		file.IsDeclarationFile,
		false,
		false,
	)
	hosts := make([]documentedHost, 0, len(inventory.Units))
	for _, unit := range inventory.Units {
		nodes := inventory.UnitNodes[unit.ID]
		if len(nodes) == 0 {
			continue
		}
		hosts = append(hosts, documentedHost{
			Node:    nodes[0],
			Nodes:   nodes,
			Symbol:  unit.Symbol,
			Targets: []string{unit.Target},
		})
	}
	sort.Slice(hosts, func(left int, right int) bool {
		return hosts[left].Node.Pos() < hosts[right].Node.Pos()
	})
	return mergeSharedBlockHosts(hosts)
}

// mergeSharedBlockHosts folds identities that can only ever share one block.
//
// `export const a = 1, b = 2;` declares two identities, and TypeScript gives
// the statement a single JSDoc position serving both. They are two obligations
// with one repair, so reporting each separately tells the author the same thing
// twice — the duplication this project's diagnostics deliberately avoid.
func mergeSharedBlockHosts(hosts []documentedHost) []documentedHost {
	merged := make([]documentedHost, 0, len(hosts))
	byStatement := map[*shimast.Node]int{}
	for _, host := range hosts {
		statement := sharedVariableStatement(host)
		if statement == nil {
			merged = append(merged, host)
			continue
		}
		if index, seen := byStatement[statement]; seen &&
			merged[index].Symbol == host.Symbol {
			merged[index].Targets = append(merged[index].Targets, host.Targets...)
			continue
		}
		byStatement[statement] = len(merged)
		merged = append(merged, host)
	}
	return merged
}

func sharedVariableStatement(host documentedHost) *shimast.Node {
	for _, node := range host.Nodes {
		if node != nil && node.Kind == shimast.KindVariableStatement {
			return node
		}
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
func jsdocState(file *shimast.SourceFile, node *shimast.Node) jsdocPresence {
	content := file.Text()
	state := jsdocMissing
	for _, doc := range node.JSDoc(file) {
		if doc == nil || doc.Pos() < 0 || doc.End() > len(content) {
			continue
		}
		if jsdocHasContent(content[doc.Pos():doc.End()]) {
			return jsdocPresent
		}
		state = jsdocEmpty
	}
	return state
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
