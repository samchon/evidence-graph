package evidence

import "testing"

/**
 * Verifies a merged interface and namespace documented on the interface half.
 *
 * One of the four positions in the merged-identity matrix. An identity spread
 * over two declarations has two places a block could sit, and the rule is that
 * exactly one of them holds it — so this is the accepted shape, not merely a
 * tolerated one.
 *
 *  1. Document only the interface half of a merged identity.
 *  2. Run the rule with the default selection.
 *  3. Assert silence.
 */
func TestDocumentedAcceptsAMergedIdentityDocumentedOnTheInterface(t *testing.T) {
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
 * Verifies the same identity documented on the namespace half instead.
 *
 * The rule counts blocks per identity, never per declaration kind, so which
 * half carries the block is the author's choice. A rule that privileged the
 * interface would force a rewrite of every file that documents the companion.
 *
 *  1. Document only the namespace half of a merged identity.
 *  2. Run the rule.
 *  3. Assert silence.
 */
func TestDocumentedAcceptsAMergedIdentityDocumentedOnTheNamespace(t *testing.T) {
	assertSilent(t, runDocumentedRule(t, "src/ISale.ts", `
export interface ISale {
  /** Identifier of the sale. */
  id: string;
}
/** A sale offered to a customer. */
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
 * Verifies two blocks on one merged identity are reported.
 *
 * The tag parser reads every block, so two blocks on one identity means a
 * citation could live in either and the identity's provenance is split across
 * two places a reviewer has to find. The diagnostic names both lines because
 * naming one would leave the reader guessing which to keep.
 *
 *  1. Document both halves of a merged identity.
 *  2. Run the rule.
 *  3. Assert one finding naming both block locations.
 */
func TestDocumentedReportsAMergedIdentityDocumentedTwice(t *testing.T) {
	messages := runDocumentedRule(t, "src/ISale.ts", `
/** A sale offered to a customer. */
export interface ISale {
  /** Identifier of the sale. */
  id: string;
}
/** Companion contracts of a sale. */
export namespace ISale {
  /** Creation input. */
  export interface ICreate {
    /** Identifier of the sale. */
    id: string;
  }
}
`, "")
	assertReportedAmong(t, messages, "Duplicate JSDoc on exported type 'ISale'")
	assertReportedAmong(t, messages, "blocks at line 2 and line 7")
}

/**
 * Verifies a merged identity documented on neither half is still missing.
 *
 * The fourth position in the matrix, and the twin that keeps the duplicate
 * branch from swallowing the zero case.
 *
 *  1. Document neither half of a merged identity.
 *  2. Run the rule.
 *  3. Assert the missing diagnostic, reported once.
 */
func TestDocumentedReportsAMergedIdentityDocumentedOnNeitherHalf(t *testing.T) {
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
 * Verifies a class merged with a namespace, documented on the class half.
 *
 * A class declaration is deliberately not a type unit, so its block is invisible
 * to the collector. Left that way, a file documenting the class half would be
 * told its identity is undocumented while the block sits directly above it.
 *
 *  1. Document only the class half of a merged identity.
 *  2. Run the rule.
 *  3. Assert silence.
 */
func TestDocumentedAcceptsAMergedClassDocumentedOnTheClass(t *testing.T) {
	assertSilent(t, runDocumentedRule(t, "src/Something.ts", `
/** The exported service. */
export class Something {}
export namespace Something {
  /** Current version. */
  export const version = "1";
}
`, ""))
}

/**
 * Verifies a class and namespace documented on both halves are reported.
 *
 * The twin of the case above: folding the class declaration into the identity
 * must make its block count, not merely make it forgivable.
 *
 *  1. Document both halves of a merged class identity.
 *  2. Run the rule.
 *  3. Assert the duplicate diagnostic.
 */
func TestDocumentedReportsAMergedClassDocumentedTwice(t *testing.T) {
	assertReportedAmong(t, runDocumentedRule(t, "src/Something.ts", `
/** The exported service. */
export class Something {}
/** Companion values of the service. */
export namespace Something {
  /** Current version. */
  export const version = "1";
}
`, ""), "Duplicate JSDoc on exported type 'Something'")
}

/**
 * Verifies a const documented beside its default export is accepted.
 *
 * `export default x` declares nothing and materializes no unit, but it is a
 * second position a block can occupy above one identity. This is the shape the
 * plugin's own entry point uses.
 *
 *  1. Document the const and leave the default export bare.
 *  2. Run the rule.
 *  3. Assert silence.
 */
func TestDocumentedAcceptsAConstDocumentedBesideItsDefaultExport(t *testing.T) {
	assertSilent(t, runDocumentedRule(t, "src/evidence.ts", `
/** The exported plugin descriptor. */
export const evidence = { name: "evidence" };
export default evidence;
`, ""))
}

/**
 * Verifies a const and its default export documented twice are reported.
 *
 * Without folding the export assignment into the identity, the second block
 * would be invisible and the duplicate rule blind to the merged form authors
 * reach for most.
 *
 *  1. Document both the const and its default export.
 *  2. Run the rule.
 *  3. Assert the duplicate diagnostic.
 */
func TestDocumentedReportsAConstAndDefaultExportDocumentedTwice(t *testing.T) {
	assertReportedAmong(t, runDocumentedRule(t, "src/evidence.ts", `
/** The exported plugin descriptor. */
export const evidence = { name: "evidence" };
/** The default export of this module. */
export default evidence;
`, ""), "Duplicate JSDoc on exported property 'evidence'")
}

/**
 * Verifies a default export of an undocumented const is still missing.
 *
 * The twin keeping the export-assignment fold from exempting the pair
 * altogether.
 *
 *  1. Leave both the const and its default export bare.
 *  2. Run the rule.
 *  3. Assert the missing diagnostic.
 */
func TestDocumentedReportsAnUndocumentedConstWithADefaultExport(t *testing.T) {
	assertReported(t, runDocumentedRule(t, "src/evidence.ts", `
export const evidence = { name: "evidence" };
export default evidence;
`, ""), "Missing JSDoc on exported property 'evidence'")
}

/**
 * Verifies two documented overload signatures are reported.
 *
 * Recorded as a deliberate consequence rather than an oversight: per-signature
 * JSDoc is a real language-service feature, and an editor shows the matching
 * signature's block at a call site. The one-block rule makes that an error, so
 * the behavior is pinned here where a reviewer can veto it on evidence.
 *
 *  1. Document two signatures of one overload set.
 *  2. Run the rule.
 *  3. Assert the duplicate diagnostic naming both blocks.
 */
func TestDocumentedReportsTwoDocumentedOverloadSignatures(t *testing.T) {
	assertReportedAmong(t, runDocumentedRule(t, "src/format.ts", `
/** Renders a string for display. */
export function format(value: string): string;
/** Renders a number for display. */
export function format(value: number): string;
export function format(value: string | number): string {
  return String(value);
}
`, ""), "Duplicate JSDoc on exported function 'format'")
}

/**
 * Verifies an empty block does not count toward the duplicate.
 *
 * Only a block with content documents, and only such a block can hold a
 * citation. Counting an empty one would push an identity into duplicate
 * territory for a comment that says nothing, and would hide the emptiness
 * finding behind a message about having too many.
 *
 *  1. Document one half of a merged identity and leave an empty block on the
 *     other.
 *  2. Run the rule.
 *  3. Assert silence.
 */
func TestDocumentedDoesNotCountEmptyBlocksTowardTheDuplicate(t *testing.T) {
	assertSilent(t, runDocumentedRule(t, "src/ISale.ts", `
/** A sale offered to a customer. */
export interface ISale {
  /** Identifier of the sale. */
  id: string;
}
/** */
export namespace ISale {
  /** Creation input. */
  export interface ICreate {
    /** Identifier of the sale. */
    id: string;
  }
}
`, ""))
}
