import {
  assertFailure,
  assertIncludes,
  createProject,
  runCheck,
  type IEvidenceProject,
} from "../internal/project.ts";

/**
 * Verifies the packaged rule fails a build when one identity carries two
 * blocks.
 *
 * The accepting twin passes because each fixture documents exactly one half,
 * not because the rule is blind to the second — and a rule that never fires is
 * indistinguishable from one that passed. This also pins the two folds the
 * collector does not do for us: a class declaration is not a unit, and an
 * export assignment declares nothing, yet a block above either counts against
 * the identity it spells.
 *
 * 1. Document both halves of an interface, a class, and a default export.
 * 2. Enable `evidence/documented` with the default selection.
 * 3. Assert a non-zero exit naming each identity and both block locations.
 */
export const test_evidence_documented_reports_duplicate_blocks = (): void => {
  const project: IEvidenceProject = createProject({
    name: "documented-duplicate",
    include: ["src"],
    lintConfig: [
      'import { evidence } from "@samchon/lint-plugin-evidence";',
      "",
      "export default {",
      '  plugins: { "evidence": evidence },',
      "  rules: {",
      '    "evidence/documented": "error",',
      "  },",
      "};",
      "",
    ].join("\n"),
    files: {
      "src/ISale.ts": [
        "/** A sale offered to a customer. */",
        "export interface ISale {",
        "  /** Identifier of the sale. */",
        "  id: string;",
        "}",
        "/** Companion contracts of a sale. */",
        "export namespace ISale {",
        "  /** Creation input. */",
        "  export interface ICreate {",
        "    /** Identifier of the sale. */",
        "    id: string;",
        "  }",
        "}",
        "",
      ].join("\n"),
      "src/Something.ts": [
        "/** The exported service. */",
        "export class Something {}",
        "/** Companion values of the service. */",
        "export namespace Something {",
        "  /** Current version. */",
        '  export const version = "1";',
        "}",
        "",
      ].join("\n"),
      "src/evidence.ts": [
        "/** The exported descriptor. */",
        'export const evidence = { name: "evidence" };',
        "/** The default export of this module. */",
        "export default evidence;",
        "",
      ].join("\n"),
    },
  });
  try {
    const result = runCheck(project.directory);
    assertFailure(
      result,
      "Two blocks on one identity must fail the build, not be tolerated.",
    );
    assertIncludes(
      result,
      "Duplicate JSDoc on exported type 'ISale'",
      "A merged interface and namespace must be reported as one identity.",
    );
    assertIncludes(
      result,
      "blocks at line 1 and line 6",
      "The diagnostic must name every block location, or the reader cannot tell which to keep.",
    );
    assertIncludes(
      result,
      "Duplicate JSDoc on exported type 'Something'",
      "A class declaration is not a unit, but a block above it still documents the identity.",
    );
    assertIncludes(
      result,
      "Duplicate JSDoc on exported property 'evidence'",
      "An export assignment declares nothing, but a block above it still documents the identity.",
    );
  } finally {
    project.cleanup();
  }
};
