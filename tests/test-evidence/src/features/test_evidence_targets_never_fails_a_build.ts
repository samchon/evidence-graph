import {
  assertExcludes,
  assertFailure,
  assertIncludes,
  createProject,
  runCheck,
  type IEvidenceProject,
} from "../internal/index.ts";

/**
 * Verifies the packaged targets rule reports nothing, even beside a graph that
 * is failing.
 *
 * The corpus itself cannot be observed from here: the host computes hints
 * through a separate entry point and never during `ttsc check`, so a feature
 * case can reach the rule's silence but not its output. Silence is the half
 * worth reaching, because it is the half the design rests on — the host offers
 * a corpus only for a project rule that passed, so a single report from this
 * rule would disable completions exactly when an author is writing the citation
 * they exist to help with.
 *
 * 1. Enable `evidence/graph` and `evidence/targets` on the same declaration.
 * 2. Leave a documented section unacknowledged, so the graph fails.
 * 3. Assert the graph's finding appears and nothing is attributed to targets.
 */
export const test_evidence_targets_never_fails_a_build = (): void => {
  const project: IEvidenceProject = createProject({
    name: "targets-silence",
    include: ["src"],
    lintConfig: [
      'import type { ITtscLintConfig } from "@ttsc/lint";',
      'import { evidence, type IEvidenceGraphConfig } from "@samchon/lint-plugin-evidence";',
      "",
      "const graph: IEvidenceGraphConfig = {",
      "  claims: [{",
      '    type: "typescript",',
      '    files: ["src/**"],',
      '    symbol: "type",',
      "    reference: {",
      '      type: "markdown",',
      '      files: ["docs/**"],',
      '      symbol: "h2",',
      "    },",
      "  }],",
      "};",
      "",
      "export default {",
      '  plugins: { "evidence": evidence },',
      "  rules: {",
      '    "evidence/graph": ["error", graph],',
      '    "evidence/targets": ["error", graph],',
      "  },",
      "} satisfies ITtscLintConfig;",
      "",
    ].join("\n"),
    files: {
      "docs/pricing.md": "## Sale Price {#sale-price}\n",
      "src/ISale.ts": [
        "/** A sale offered to a customer. */",
        "export interface ISale {",
        "  /** Price of the sale. */",
        "  price: number;",
        "}",
        "",
      ].join("\n"),
    },
  });
  try {
    const result = runCheck(project.directory);
    assertFailure(
      result,
      "The unacknowledged section must still fail the build through evidence/graph.",
    );
    assertIncludes(
      result,
      "Missing acknowledgement",
      "The graph must report the unmet obligation as it always has.",
    );
    assertExcludes(
      result,
      "evidence/targets",
      "The targets rule must report nothing; a single finding would cost it the pass its corpus depends on.",
    );
  } finally {
    project.cleanup();
  }
};
