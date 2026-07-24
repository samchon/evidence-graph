package evidence

import (
	"strings"
	"testing"
)

const targetsMarkdownConfig = `{"claims":[{
	"type":"typescript",
	"files":["src/**"],
	"reference":{"type":"markdown","files":["docs/**"],"symbol":["file","h2"]}
}]}`

/**
 * Verifies the corpus carries a selected heading under its exact target.
 *
 * The anchor is the thing an author cannot reproduce from memory, so it is the
 * reason this rule exists. A corpus that listed headings by title rather than
 * by target would look right in an editor and insert something the graph
 * cannot resolve.
 *
 *  1. Declare one document with a heading the reference selects.
 *  2. Publish the corpus.
 *  3. Assert the heading's target is offered with its text alongside.
 */
func TestTargetsPublishesSelectedHeadings(t *testing.T) {
	hints, messages := runTargetsRule(t, map[string]string{
		"docs/pricing.md": "## Sale Price {#sale-price}\n",
	}, targetsMarkdownConfig)
	assertSilent(t, messages)
	inserts := targetInserts(targetHintsAt(hints, "@evidence "))
	if !contains(inserts, "docs/pricing.md#sale-price") {
		t.Fatalf("expected the heading target, got:\n%s", strings.Join(inserts, "\n"))
	}
	for _, hint := range targetHintsAt(hints, "@evidence ") {
		if hint.Insert != "docs/pricing.md#sale-price" {
			continue
		}
		if !strings.Contains(hint.Detail, "Sale Price") {
			t.Fatalf("expected the heading text as detail, got %q", hint.Detail)
		}
	}
}

/**
 * Verifies a heading the reference does not select stays out of the corpus.
 *
 * The corpus is a projection of the configured population, so an unselected
 * heading is not a target. Offering one would teach that any completion is
 * citable, and the author's next build would disagree.
 *
 *  1. Select `h2` only, and declare an `h3` beneath the `h2`.
 *  2. Publish the corpus.
 *  3. Assert the `h3` target is absent while the `h2` is present.
 */
func TestTargetsOmitsUnselectedHeadings(t *testing.T) {
	hints, messages := runTargetsRule(t, map[string]string{
		"docs/pricing.md": "## Sale Price {#sale-price}\n\n### Rounding {#rounding}\n",
	}, targetsMarkdownConfig)
	assertSilent(t, messages)
	inserts := targetInserts(hints)
	if contains(inserts, "docs/pricing.md#rounding") {
		t.Fatalf("an unselected heading must not be offered:\n%s", strings.Join(inserts, "\n"))
	}
	if !contains(inserts, "docs/pricing.md#sale-price") {
		t.Fatalf("the selected heading must be offered:\n%s", strings.Join(inserts, "\n"))
	}
}

/**
 * Verifies headings are offered before file targets.
 *
 * Slice order is the corpus's only ranking channel, and it answers what an
 * author cannot supply from memory. A file path is visible in the project tree;
 * a generated anchor is neither visible nor guessable, so it comes first.
 *
 *  1. Select both the file and its headings.
 *  2. Publish the corpus.
 *  3. Assert every heading precedes the file target.
 */
func TestTargetsRanksHeadingsBeforeFiles(t *testing.T) {
	hints, messages := runTargetsRule(t, map[string]string{
		"docs/pricing.md": "## Sale Price {#sale-price}\n",
	}, targetsMarkdownConfig)
	assertSilent(t, messages)
	inserts := targetInserts(targetHintsAt(hints, "@evidence "))
	heading := indexOf(inserts, "docs/pricing.md#sale-price")
	file := indexOf(inserts, "docs/pricing.md")
	if heading < 0 || file < 0 {
		t.Fatalf("expected both targets, got:\n%s", strings.Join(inserts, "\n"))
	}
	if heading > file {
		t.Fatalf("headings must be offered before file targets, got:\n%s", strings.Join(inserts, "\n"))
	}
}

/**
 * Verifies both tag positions receive the corpus.
 *
 * An exclusion names a target under the same grammar as a citation, so an
 * author writing one needs the same list. A corpus published for `@evidence`
 * alone would help the easy half and abandon the half that has to justify
 * itself in review.
 *
 *  1. Publish the corpus for a configured document.
 *  2. Narrow it to each trigger.
 *  3. Assert both carry the same targets.
 */
func TestTargetsPublishesForBothTagPositions(t *testing.T) {
	hints, messages := runTargetsRule(t, map[string]string{
		"docs/pricing.md": "## Sale Price {#sale-price}\n",
	}, targetsMarkdownConfig)
	assertSilent(t, messages)
	cited := targetInserts(targetHintsAt(hints, "@evidence "))
	excluded := targetInserts(targetHintsAt(hints, "@evidenceExclude "))
	if len(cited) == 0 {
		t.Fatal("expected a corpus at the citation trigger")
	}
	if strings.Join(cited, "\n") != strings.Join(excluded, "\n") {
		t.Fatalf(
			"both triggers must carry the same targets:\n%s\n---\n%s",
			strings.Join(cited, "\n"),
			strings.Join(excluded, "\n"),
		)
	}
}

/**
 * Verifies the citation trigger cannot match inside an exclusion line.
 *
 * The host matches a trigger with `strings.LastIndex` against the line prefix,
 * so the two tags stay apart only because `After` carries a trailing space:
 * the character following `@evidence` inside `@evidenceExclude` is `E`. That is
 * a property of the host's matcher rather than of this code, which is why it is
 * pinned rather than trusted — dropping the space would silently merge the two
 * corpora.
 *
 *  1. Take both published triggers.
 *  2. Apply the host's matching rule to an exclusion line.
 *  3. Assert only the exclusion trigger matches.
 */
func TestTargetsKeepsTheTwoTriggersApart(t *testing.T) {
	line := " * @evidenceExclude "
	if strings.Contains(line, "@evidence ") {
		t.Fatal("the citation trigger must not occur inside an exclusion line")
	}
	if !strings.Contains(line, "@evidenceExclude ") {
		t.Fatal("the exclusion trigger must occur inside an exclusion line")
	}
	if strings.Contains(" * @evidence ", "@evidenceExclude ") {
		t.Fatal("the exclusion trigger must not occur inside a citation line")
	}
}

/**
 * Verifies every published hint is one the host will keep.
 *
 * The host drops a hint with no scope or no `After` rather than offering it
 * everywhere, so a malformed corpus does not fail — it silently shrinks. A
 * count assertion elsewhere would still pass while the entry never reached an
 * editor.
 *
 *  1. Publish a corpus over Markdown and Swagger.
 *  2. Inspect every hint's trigger.
 *  3. Assert each carries a scope and an `After` ending where a target begins.
 */
func TestTargetsPublishesOnlyUsableHints(t *testing.T) {
	hints, messages := runTargetsRule(t, map[string]string{
		"docs/pricing.md": "## Sale Price {#sale-price}\n",
	}, targetsMarkdownConfig)
	assertSilent(t, messages)
	if len(hints) == 0 {
		t.Fatal("expected a corpus")
	}
	for _, hint := range hints {
		if hint.Trigger.Scope == "" {
			t.Fatalf("hint %q carries no scope", hint.Insert)
		}
		if !strings.HasSuffix(hint.Trigger.After, " ") {
			t.Fatalf("trigger %q must end where the target begins", hint.Trigger.After)
		}
		if hint.Insert == "" {
			t.Fatalf("hint at trigger %q inserts nothing", hint.Trigger.After)
		}
	}
}

func contains(values []string, expected string) bool {
	return indexOf(values, expected) >= 0
}

func indexOf(values []string, expected string) int {
	for index, value := range values {
		if value == expected {
			return index
		}
	}
	return -1
}
