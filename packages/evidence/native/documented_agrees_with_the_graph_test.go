package evidence

import "testing"

/**
 * Verifies the rule demands the block where a citation can actually live.
 *
 * A class is not a type unit, so `class Sale` beside `namespace Sale`
 * materializes the `Sale` type from the namespace alone and the collector
 * registers no host for the class. Demanding the block above the class would
 * send an author's `@evidence` into a position `evidence/graph` rejects as an
 * unsupported host — this rule steering citations somewhere a citation cannot
 * live, which is the failure it exists to prevent.
 *
 *  1. Document only the class half of a merged class identity.
 *  2. Run the rule.
 *  3. Assert the identity is still reported, because the class hosts nothing.
 */
func TestDocumentedRejectsABlockOnANonHostingDeclaration(t *testing.T) {
	assertReported(t, runDocumentedRule(t, "src/Sale.ts", `
/** A sale offered to a customer. */
export class Sale {
  price: number = 0;
}
export namespace Sale {
  /** Current version. */
  export const version = "1";
}
`, ""), "Missing JSDoc on exported type 'Sale'")
}

/**
 * Verifies the namespace half satisfies the same pair.
 *
 * The twin of the case above, and the position the rule now names. Together
 * they pin which declaration of the pair is demanded rather than leaving it to
 * be rediscovered from the collector's unit model.
 *
 *  1. Document only the namespace half of a merged class identity.
 *  2. Run the rule.
 *  3. Assert silence.
 */
func TestDocumentedAcceptsABlockOnTheHostingDeclaration(t *testing.T) {
	assertSilent(t, runDocumentedRule(t, "src/Sale.ts", `
export class Sale {
  price: number = 0;
}
/** A sale offered to a customer. */
export namespace Sale {
  /** Current version. */
  export const version = "1";
}
`, ""))
}

/**
 * Verifies `evidence/graph` accepts a citation in the position this rule
 * demands.
 *
 * The two cases above prove which declaration is named; this proves the naming
 * is worth obeying. Without it the rules could agree on a position that the
 * graph then refuses, and each rule's own suite would stay green while an
 * author following one diagnostic was handed another.
 *
 *  1. Cite a Markdown section from the namespace half of a merged class.
 *  2. Run the graph with a claim selecting `type` hosts.
 *  3. Assert no diagnostic at all.
 */
func TestGraphAcceptsEvidenceOnTheDeclarationDocumentedDemands(t *testing.T) {
	assertNoProblems(t, runIndexRule(t, map[string]string{
		"docs/spec.md": "## Contract {#contract}\n",
		"src/Sale.ts": `
export class Sale {
  price: number = 0;
}
/** @evidence docs/spec.md#contract The namespace half documents this contract. */
export namespace Sale {
  /** Current version. */
  export const version = "1";
}
`,
	}, `{"claims":[{
		"type":"typescript",
		"files":["src/Sale.ts"],
		"symbol":"type",
		"reference":{"type":"markdown","files":["docs/spec.md"],"symbol":"h2"}
	}]}`))
}

/**
 * Verifies the graph rejects a citation on the declaration this rule refuses.
 *
 * The negative twin that makes the agreement falsifiable. If the class ever
 * became a host, this case fails and the narrowing above should be revisited
 * rather than silently kept.
 *
 *  1. Cite the same section from the class half instead.
 *  2. Run the graph with the same claim.
 *  3. Assert the out-of-scope host diagnostic.
 */
func TestGraphRejectsEvidenceOnTheDeclarationDocumentedRefuses(t *testing.T) {
	assertProblemContains(t, runIndexRule(t, map[string]string{
		"docs/spec.md": "## Contract {#contract}\n",
		"src/Sale.ts": `
/** @evidence docs/spec.md#contract The class half cannot host this. */
export class Sale {
  price: number = 0;
}
export namespace Sale {
  /** Current version. */
  export const version = "1";
}
`,
	}, `{"claims":[{
		"type":"typescript",
		"files":["src/Sale.ts"],
		"symbol":"type",
		"reference":{"type":"markdown","files":["docs/spec.md"],"symbol":"h2"}
	}]}`), "unsupported or non-exported declaration")
}
