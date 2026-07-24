import type { IEvidenceDocumentedConfig } from "../structures/IEvidenceDocumentedConfig";
import type { IEvidenceGraphConfig } from "../structures/IEvidenceGraphConfig";

declare module "@ttsc/lint" {
  interface ITtscLintRuleOptionsMap {
    /**
     * Declares this project's evidence graph.
     *
     * The claims define the citing populations and the independently complete
     * evidence references each one must acknowledge.
     */
    "evidence/graph": IEvidenceGraphConfig;

    /**
     * Requires a JSDoc block on every selected export.
     *
     * A JSDoc block is the only place an `@evidence` tag is read from, so an
     * export without one cannot participate in the graph at all.
     */
    "evidence/documented": IEvidenceDocumentedConfig;

    /**
     * Publishes the evidence targets an editor may complete.
     *
     * Takes the same graph declaration `evidence/graph` takes, so the value is
     * declared once and passed twice. It reports nothing under any input: the
     * host offers completions only for a project rule that passed, and a rule
     * that reported would fall silent exactly when an author is writing the
     * citation the completions exist to help with.
     */
    "evidence/targets": IEvidenceGraphConfig;
  }
}
