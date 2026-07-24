package evidence

import (
	"os"
	"sort"

	"github.com/samchon/ttsc/packages/lint/rule"
)

// targetsRule publishes the evidence targets an editor may complete.
//
// It exists as a second rule because a reporting rule cannot publish a corpus.
// The host offers hints only for a project rule that passed, and
// `ProjectContext.Report` marks a rule failed unconditionally, so a corpus
// published by `evidence/graph` would be available exactly when every
// obligation is already met — and would disappear the moment a new document
// section created the need to write a citation. This rule therefore reports
// nothing under any input; that silence is its contract, not an oversight.
type targetsRule struct{}

// targetsState is what Check publishes for Hints to project.
//
// It carries the resolved root and the decoded configuration rather than the
// loaded inventories, because Check runs during `ttsc check` and Hints does
// not. Loading here would make every build pay for a Swagger spawn to serve an
// editor that was not asking.
type targetsState struct {
	Root   string
	Config graphConfig
}

func (targetsRule) Name() string { return targetsRuleName }

func (targetsRule) NeedsTypeChecker() bool { return false }

func (targetsRule) Check(ctx *rule.ProjectContext) {
	if ctx == nil {
		return
	}
	config, problems := decodeGraphConfig(ctx.Options)
	if len(problems) != 0 {
		// Deliberately silent. A broken configuration publishes no state, which
		// the host reads as an absent corpus, and `evidence/graph` is already
		// the rule that tells the author what is wrong with it. Reporting here
		// would say it twice and cost this rule the pass it needs.
		return
	}
	root := evidenceProjectRoot(ctx.Identity)
	if root == "" {
		return
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return
	}
	ctx.SetState(targetsState{Root: root, Config: config})
}

// Hints projects the configured evidence population into a completion corpus.
//
// Loading happens here rather than in Check because the host never asks for
// hints during `ttsc check`; the cost lands only on an editor request.
func (targetsRule) Hints(ctx *rule.HintContext) []rule.Hint {
	if ctx == nil {
		return nil
	}
	state, published := ctx.State.(targetsState)
	if !published || state.Root == "" {
		return nil
	}
	markdown, _ := loadMarkdownInventories(state.Root, state.Config)
	swagger, _ := loadSwaggerInventories(state.Root, state.Config)
	units := selectedCompletionUnits(state.Config, markdown, swagger)
	routes := selectsTypeScriptReference(state.Config)
	hints := make([]rule.Hint, 0, (len(units)+1)*len(evidenceHintTriggers))
	for _, trigger := range evidenceHintTriggers {
		if routes {
			hints = append(hints, typeScriptRouteHint(trigger))
		}
		for _, unit := range units {
			hints = append(hints, rule.Hint{
				Insert:  unit.Target,
				Detail:  unit.Readable,
				Trigger: trigger,
			})
		}
	}
	return hints
}

// typeScriptRouteHint routes the author into TypeScript's own completion.
//
// The host merges a corpus into the upstream response and cannot remove from
// it, so narrowing TypeScript's symbol list is not available to us. Putting the
// author inside that list is: once the brace opens, the language service
// answers every keystroke after it, against the import scope at the cursor —
// which a corpus built once per Program could never know.
//
// The inserted text is deliberately unclosed. `Insert` is verbatim with no
// snippet expansion, so the cursor lands where the text ends; `{@link }` would
// park it one character past the only position where completion fires. A
// missing brace is visible and one keystroke, and the target grammar catches it
// if forgotten. A cursor in the wrong place is silent.
func typeScriptRouteHint(trigger rule.HintTrigger) rule.Hint {
	return rule.Hint{
		Insert:  "{@link ",
		Label:   "{@link",
		Detail:  "TypeScript symbol, resolved through this file's imports",
		Trigger: trigger,
	}
}

// selectsTypeScriptReference reports whether any claim cites TypeScript.
//
// The entry is withheld otherwise, because a graph citing only Markdown cannot
// resolve an inline-link target — offering the grammar there would hand the
// author an unresolved-target diagnostic for taking a suggestion.
//
// The condition is necessarily global. A corpus answers a keystroke with no
// file context, so it cannot tell which claim owns the file the cursor is in;
// a repository mixing a TypeScript-citing claim with a Markdown-only one offers
// the entry in both. Narrowing that needs a per-file corpus, which the contract
// deliberately does not have.
func selectsTypeScriptReference(config graphConfig) bool {
	for _, claim := range config.Claims {
		for _, reference := range claim.References {
			if reference.Type == artifactTypeScript {
				return true
			}
		}
	}
	return false
}

func init() { rule.RegisterProject(targetsRule{}) }

// evidenceHintTriggers names both tag positions a target can be written in.
//
// Each `After` ends where the target begins, which is what makes the text the
// author has typed so far the editor's filter. The trailing space also keeps
// the two apart: `"@evidence "` cannot occur inside `"@evidenceExclude "`,
// because the character following `@evidence` there is `E`.
var evidenceHintTriggers = []rule.HintTrigger{
	{Scope: rule.HintScopeJSDoc, After: "@evidence "},
	{Scope: rule.HintScopeJSDoc, After: "@evidenceExclude "},
}

// selectedCompletionUnits collects the units some configured reference selects,
// in the order they should be offered.
//
// Slice order is the corpus's only ranking channel, so it answers what an
// author cannot supply from memory. A heading's generated anchor is neither
// visible in the project tree nor guessable from its text, so selected headings
// come first; a Markdown file path is typed as easily as it is read, so file
// targets follow; Swagger operations come last, being both few and mechanical.
//
// TypeScript units are absent on purpose. TypeScript's own language service
// completes symbols inside `{@link}` against the scope at the cursor, which a
// corpus built once per Program cannot know — offering one here would duplicate
// a correct list with a worse one.
func selectedCompletionUnits(
	config graphConfig,
	markdown map[string]*artifactInventory,
	swagger map[string]*artifactInventory,
) []*evidenceUnit {
	ranked := map[string][]*evidenceUnit{}
	seen := map[string]bool{}
	for _, claim := range config.Claims {
		for _, reference := range claim.References {
			inventories := inventoriesOf(
				reference.Type,
				markdown,
				swagger,
				map[string]*artifactInventory{},
			)
			for _, path := range matchingReferencePaths(inventories, reference) {
				inventory := inventories[path]
				if inventory == nil {
					continue
				}
				for _, unit := range inventory.Units {
					if !reference.Symbols.contains(unit.Symbol) || seen[unit.ID] {
						continue
					}
					seen[unit.ID] = true
					group := completionRank(unit)
					ranked[group] = append(ranked[group], unit)
				}
			}
		}
	}
	units := []*evidenceUnit{}
	for _, group := range []string{"heading", "file", "operation"} {
		tier := ranked[group]
		sort.SliceStable(tier, func(left int, right int) bool {
			if tier[left].Path != tier[right].Path {
				return tier[left].Path < tier[right].Path
			}
			return tier[left].Line < tier[right].Line
		})
		units = append(units, tier...)
	}
	return units
}

func completionRank(unit *evidenceUnit) string {
	switch {
	case unit.Type == artifactSwagger:
		return "operation"
	case unit.Symbol == "file":
		return "file"
	default:
		return "heading"
	}
}
