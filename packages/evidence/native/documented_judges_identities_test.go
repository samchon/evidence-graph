package evidence

import "testing"

/**
 * Verifies a merged interface and namespace need one block between them.
 *
 * This is the idiom the whole rule set is built around, and `evidence/singular`
 * blesses it explicitly. A rule that judged declaration nodes rather than
 * identities would demand a second block on the namespace half, which no author
 * writes and no reviewer would accept.
 *
 *  1. Document an interface and its companion namespace once, on the interface.
 *  2. Run the rule with the default selection.
 *  3. Assert silence.
 */
func TestDocumentedAcceptsOneBlockForAMergedInterfaceAndNamespace(t *testing.T) {
	assertSilent(t, runDocumentedRule(t, "src/ISale.ts", `
/** A sale offered to a customer. */
export interface ISale {
  /** Identifier of the sale. */
  id: string;
}
export namespace ISale {
  /** Creation input. */
  export interface ICreate {
    /** Identifier of the sale. */
    id: string;
  }
}
`, ""))
}

/**
 * Verifies the same for a class merged with a namespace.
 *
 * A class declaration is not itself a host, so the namespace half carries the
 * only `type` unit of that name — and it must still be discharged by the block
 * a reader would write once.
 *
 *  1. Document a class and its companion namespace once.
 *  2. Run the rule.
 *  3. Assert silence.
 */
func TestDocumentedAcceptsOneBlockForAMergedClassAndNamespace(t *testing.T) {
	assertSilent(t, runDocumentedRule(t, "src/Something.ts", `
/** The exported service. */
export class Something {}
/** The exported service. */
export namespace Something {
  /** Current version. */
  export const version = "1";
}
`, ""))
}

/**
 * Verifies the merge did not become "never report a namespace".
 *
 * The negative twin of the two cases above. Folding merged declarations into
 * one identity must not exempt a namespace that is genuinely undocumented, or
 * the fix would trade a false positive for a silent miss.
 *
 *  1. Leave a standalone namespace undocumented.
 *  2. Run the rule.
 *  3. Assert it is reported.
 */
func TestDocumentedStillReportsAnUndocumentedNamespace(t *testing.T) {
	assertReported(t, runDocumentedRule(t, "src/Orders.ts", `
export namespace Orders {
  /** Current version. */
  export const version = "1";
}
`, ""), "Missing JSDoc on exported type 'Orders'")
}

/**
 * Verifies an undocumented merged identity is reported once, not per half.
 *
 * The other direction of the same grouping: two declarations of one name are
 * one obligation, so they owe one finding.
 *
 *  1. Leave both halves of a merged identity undocumented.
 *  2. Run the rule.
 *  3. Assert a single finding.
 */
func TestDocumentedReportsAMergedIdentityOnce(t *testing.T) {
	assertReported(t, runDocumentedRule(t, "src/ISale.ts", `
export interface ISale {
  /** Identifier of the sale. */
  id: string;
}
export namespace ISale {
  /** Current version. */
  export const version = "1";
}
`, ""), "Missing JSDoc on exported type 'ISale'")
}

/**
 * Verifies a documented inner declaration does not discharge an outer one of
 * the same name.
 *
 * Identity is qualified, so `A.f` and `f` are different obligations. A grouping
 * keyed on the bare name plus source adjacency would let the inner block excuse
 * the outer declaration — a silent miss, and the failure mode this rule exists
 * to remove.
 *
 *  1. Document a namespace member named `f` and leave a top-level `f` bare.
 *  2. Run the rule.
 *  3. Assert the top-level declaration is still reported.
 */
func TestDocumentedKeepsQualifiedIdentitiesSeparate(t *testing.T) {
	assertReported(t, runDocumentedRule(t, "src/f.ts", `
/** Namespace A. */
export namespace A {
  /** Inner documented. */
  export function f(): void {}
}
export function f(): void {}
`, ""), "Missing JSDoc on exported function 'f'")
}

/**
 * Verifies a property is named with its owner.
 *
 * The graph addresses those units as `IAlpha.id` and `IBeta.id`, so a finding
 * naming a bare `id` twice leaves a reader unable to tell the two apart in a
 * build log — and unable to write the citation the diagnostic is asking for.
 *
 *  1. Leave a same-named property undocumented on two interfaces.
 *  2. Run the rule.
 *  3. Assert each finding carries its owner.
 */
func TestDocumentedNamesPropertiesWithTheirOwner(t *testing.T) {
	messages := runDocumentedRule(t, "src/contracts.ts", `
/** First contract. */
export interface IAlpha {
  id: string;
}
/** Second contract. */
export interface IBeta {
  id: string;
}
`, "")
	assertReportedAmong(t, messages, "exported property 'IAlpha.id'")
	assertReportedAmong(t, messages, "exported property 'IBeta.id'")
}

/**
 * Verifies a namespace member is named with its namespace.
 *
 * Same reasoning one scope deeper: `Orders.version` is the target a citation
 * would have to name.
 *
 *  1. Leave a namespace member undocumented.
 *  2. Run the rule.
 *  3. Assert the qualified name is reported.
 */
func TestDocumentedNamesNamespaceMembersWithTheirScope(t *testing.T) {
	assertReported(t, runDocumentedRule(t, "src/Orders.ts", `
/** Order contracts. */
export namespace Orders {
  export const version = "1";
}
`, ""), "exported property 'Orders.version'")
}
