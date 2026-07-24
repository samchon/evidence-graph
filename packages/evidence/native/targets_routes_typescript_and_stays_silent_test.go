package evidence

import (
	"strings"
	"testing"
)

const targetsTypeScriptConfig = `{"claims":[{
	"type":"markdown",
	"files":["docs/**"],
	"reference":{"type":"typescript","files":["src/**"],"symbol":"type"}
}]}`

/**
 * Verifies the inline-link entry is offered, unclosed, and first.
 *
 * We cannot narrow TypeScript's completion list — the host merges into the
 * upstream response and never removes from it — so routing the author into it
 * is the only move left. The unclosed form is what makes the routing work: the
 * cursor lands where the language service fires, because `Insert` is verbatim
 * and has no snippet expansion to place it anywhere else.
 *
 *  1. Configure a claim citing a TypeScript reference.
 *  2. Publish the corpus.
 *  3. Assert the entry leads, and inserts `{@link ` with its trailing space.
 */
func TestTargetsRoutesIntoTypeScriptCompletion(t *testing.T) {
	hints, messages := runTargetsRule(t, map[string]string{
		"docs/pricing.md": "## Sale Price {#sale-price}\n",
	}, targetsTypeScriptConfig)
	assertSilent(t, messages)
	cited := targetHintsAt(hints, "@evidence ")
	if len(cited) == 0 {
		t.Fatal("expected a corpus at the citation trigger")
	}
	if cited[0].Insert != "{@link " {
		t.Fatalf("expected the unclosed inline-link opener first, got %q", cited[0].Insert)
	}
	if strings.Contains(cited[0].Insert, "}") {
		t.Fatalf("a closed form parks the cursor past the completion, got %q", cited[0].Insert)
	}
}

/**
 * Verifies the inline-link entry is withheld when nothing cites TypeScript.
 *
 * A graph citing only Markdown cannot resolve an inline-link target, so
 * offering the grammar there hands the author an unresolved-target diagnostic
 * for taking a suggestion. The entry is a projection of the configuration, not
 * a constant.
 *
 *  1. Configure a claim whose only reference is Markdown.
 *  2. Publish the corpus.
 *  3. Assert no entry inserts the opener.
 */
func TestTargetsWithholdsTheLinkEntryWithoutATypeScriptReference(t *testing.T) {
	hints, messages := runTargetsRule(t, map[string]string{
		"docs/pricing.md": "## Sale Price {#sale-price}\n",
	}, targetsMarkdownConfig)
	assertSilent(t, messages)
	for _, hint := range hints {
		if strings.HasPrefix(hint.Insert, "{@link") {
			t.Fatalf("the inline-link entry must be withheld, got %q", hint.Insert)
		}
	}
}

/**
 * Verifies no TypeScript unit is offered as its own entry.
 *
 * TypeScript's language service already lists exactly the symbols in scope at
 * the cursor. A corpus built once per Program cannot know that scope, so an
 * entry per symbol would duplicate a correct list with a worse one — and would
 * survive as a stale suggestion after the symbol moved.
 *
 *  1. Configure a TypeScript reference over a real exported type.
 *  2. Publish the corpus.
 *  3. Assert the type's name is offered by nothing but the opener.
 */
func TestTargetsOffersNoTypeScriptSymbols(t *testing.T) {
	hints, messages := runTargetsRule(t, map[string]string{
		"docs/pricing.md": "## Sale Price {#sale-price}\n",
		"src/ISale.ts":    "export interface ISale {\n  price: number;\n}\n",
	}, targetsTypeScriptConfig)
	assertSilent(t, messages)
	for _, hint := range hints {
		if hint.Insert == "ISale" {
			t.Fatal("a TypeScript symbol must not be offered as its own entry")
		}
	}
}

/**
 * Verifies the rule stays silent on a graph that would fail.
 *
 * This is the property the whole design rests on. The host offers a corpus only
 * for a project rule that passed, and `Report` marks a rule failed
 * unconditionally, so a rule that reported here would fall silent exactly when
 * an author is writing the citation the corpus exists to help with.
 *
 *  1. Configure a document with a heading nothing acknowledges.
 *  2. Run the rule that would report it, and this one, over the same project.
 *  3. Assert the graph reports and this rule does not, while still publishing.
 */
func TestTargetsStaysSilentWhereTheGraphFails(t *testing.T) {
	files := map[string]string{
		"docs/pricing.md": "## Sale Price {#sale-price}\n",
		"src/sale.ts":     "export interface ISale {\n  price: number;\n}\n",
	}
	graphMessages := runIndexRule(t, files, targetsMarkdownConfig)
	if len(graphMessages) == 0 {
		t.Fatal("expected the graph to report an unmet obligation for this fixture")
	}
	hints, messages := runTargetsRule(t, files, targetsMarkdownConfig)
	assertSilent(t, messages)
	if len(hints) == 0 {
		t.Fatal("the corpus must survive a graph that fails; that is why it is a separate rule")
	}
}

/**
 * Verifies an undecodable configuration publishes nothing and says nothing.
 *
 * Reporting here would cost the rule the pass its corpus depends on, and would
 * say twice what `evidence/graph` already says once. Publishing no state is how
 * the host is told there is no corpus.
 *
 *  1. Configure the rule with a malformed options object.
 *  2. Run it.
 *  3. Assert an empty corpus and no findings.
 */
func TestTargetsPublishesNothingForAnUndecodableConfiguration(t *testing.T) {
	hints, messages := runTargetsRule(t, map[string]string{
		"docs/pricing.md": "## Sale Price {#sale-price}\n",
	}, `{"claims":[{"type":"nonsense","files":["docs/**"]}]}`)
	assertSilent(t, messages)
	if len(hints) != 0 {
		t.Fatalf("expected no corpus, got %d hints", len(hints))
	}
}

/**
 * Verifies Swagger operations rank last and carry their readable form.
 *
 * Selection is exercised directly rather than through the loader, because
 * materializing a Swagger document spawns Node and this assertion is about
 * ranking rather than about parsing. The loader has its own coverage.
 *
 *  1. Build one Markdown heading and one Swagger operation.
 *  2. Project the corpus population.
 *  3. Assert the operation is offered after the heading.
 */
func TestTargetsRanksSwaggerOperationsLast(t *testing.T) {
	config, problems := decodeGraphConfig([]byte(`{"claims":[{
		"type":"typescript",
		"files":["src/**"],
		"reference":[
			{"type":"markdown","files":["docs/**"],"symbol":["h2"]},
			{"type":"swagger","file":"openapi.json"}
		]
	}]}`))
	if len(problems) != 0 {
		t.Fatalf("fixture configuration must decode, got:\n%s", strings.Join(problems, "\n"))
	}
	markdown := map[string]*artifactInventory{
		"docs/pricing.md": {
			Path: "docs/pricing.md",
			Type: artifactMarkdown,
			Units: []*evidenceUnit{{
				ID:       "markdown:docs/pricing.md:#sale-price",
				Target:   "docs/pricing.md#sale-price",
				Type:     artifactMarkdown,
				Symbol:   "h2",
				Path:     "docs/pricing.md",
				Line:     1,
				Readable: "Markdown H2 'Sale Price'",
			}},
		},
	}
	swagger := map[string]*artifactInventory{
		"openapi.json": {
			Path: "openapi.json",
			Type: artifactSwagger,
			Units: []*evidenceUnit{{
				ID:       "swagger:openapi.json:POST:/members",
				Target:   "POST:/members",
				Type:     artifactSwagger,
				Symbol:   "operation",
				Path:     "openapi.json",
				Readable: "Swagger operation 'POST /members'",
			}},
		},
	}
	units := selectedCompletionUnits(config, markdown, swagger)
	targets := make([]string, 0, len(units))
	for _, unit := range units {
		targets = append(targets, unit.Target)
	}
	want := "docs/pricing.md#sale-price\nPOST:/members"
	if strings.Join(targets, "\n") != want {
		t.Fatalf("corpus order:\n%s\nwant:\n%s", strings.Join(targets, "\n"), want)
	}
}
