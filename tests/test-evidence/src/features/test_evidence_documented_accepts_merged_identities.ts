import {
  assertExcludes,
  assertStatus,
  createProject,
  runCheck,
  type IEvidenceProject,
} from "../internal/project.ts";

/**
 * Verifies the packaged rule accepts one block per merged identity.
 *
 * This is the idiom a consumer meets first, and the one `evidence/singular`
 * blesses by name. A rule judging declaration nodes instead of identities
 * demands a second block on the namespace half, which no author writes — so the
 * two rules would contradict each other on the very shape they were built
 * around. Driving it through the real binary is what proves the agreement
 * survives packaging.
 *
 * 1. Declare an interface, a class, and a callable, each merged with a same-named
 *    namespace and documented once.
 * 2. Enable `evidence/documented` with the default selection.
 * 3. Assert a clean exit.
 */
export const test_evidence_documented_accepts_merged_identities = (): void => {
  const project: IEvidenceProject = createProject({
    name: "documented-merged",
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
      "src/format.ts": [
        "/** Renders a value for display. */",
        "export function format(value: string): string;",
        "export function format(value: number): string;",
        "export function format(value: string | number): string {",
        "  return String(value);",
        "}",
        "",
      ].join("\n"),
    },
  });
  try {
    const result = runCheck(project.directory);
    assertStatus(
      result,
      0,
      "One block must document a whole merged identity, matching evidence/singular.",
    );
    assertExcludes(
      result,
      "evidence/documented",
      "No half of a merged identity may be reported separately.",
    );
  } finally {
    project.cleanup();
  }
};
