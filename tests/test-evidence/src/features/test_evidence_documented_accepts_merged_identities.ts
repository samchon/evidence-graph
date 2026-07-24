import {
  assertExcludes,
  assertStatus,
  createProject,
  runCheck,
  type IEvidenceProject,
} from "../internal/project.ts";

/**
 * Verifies the packaged rule accepts exactly one block per merged identity.
 *
 * This is the idiom a consumer meets first, and the one `evidence/singular`
 * blesses by name. Each fixture carries its block on a different half on
 * purpose: the rule counts blocks per identity, never per declaration kind, so
 * which half holds it is the author's choice. Driving that through the real
 * binary is what proves the agreement between the two rules survives
 * packaging.
 *
 * 1. Declare an interface, a class, and an overload set, each merged with a
 *    same-named namespace or signature run and documented on exactly one half.
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
        "export class Something {}",
        "/** The exported service. */",
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
      "src/evidence.ts": [
        "/** The exported descriptor. */",
        'export const evidence = { name: "evidence" };',
        "export default evidence;",
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
