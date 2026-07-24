/// <reference types="node" />

import type { ITtscLintPlugin } from "@ttsc/lint";
import path from "node:path";
import type { IEvidenceGraphConfig } from "./structures/index";
import { version } from "../package.json";

export * from "./structures/index";
export * from "./typings/index";

/**
 * The `@ttsc/lint` contributor that checks a project's evidence graph.
 *
 * Import this value into `lint.config.ts` and register it under the
 * `"evidence"` plugin name. You can then enable `"evidence/graph"` and pass an
 * {@link IEvidenceGraphConfig} that describes which documents and TypeScript
 * symbols must remain connected.
 *
 * The plugin contributes two rules, enabled independently.
 *
 * - `"evidence/graph"` — the configured evidence graph. Every declaration target
 *   must resolve, and every selected evidence unit must be acknowledged. Takes
 *   an {@link IEvidenceGraphConfig}.
 * - `"evidence/singular"` — one public identity per TypeScript file, named after
 *   the file. Takes no options, so it carries a bare severity.
 *
 * @example <caption>Configure the plugin in `lint.config.ts`</caption>
 *   import type { ITtscLintConfig } from "@ttsc/lint";
 *   import {
 *     evidence,
 *     type IEvidenceGraphConfig,
 *   } from "@samchon/lint-plugin-evidence";
 *
 *   const graph: IEvidenceGraphConfig = {
 *     claims: [
 *       {
 *         type: "typescript",
 *         files: ["src/**"],
 *         reference: {
 *           type: "markdown",
 *           files: ["docs/*.md"],
 *         },
 *       },
 *     ],
 *   };
 *
 *   export default {
 *     plugins: {
 *       evidence: evidence,
 *     },
 *     files: ["src/**"],
 *     rules: {
 *       "evidence/graph": ["error", graph],
 *       "evidence/singular": "error",
 *     },
 *   } satisfies ITtscLintConfig;
 */
export const evidence = {
  meta: {
    name: "@samchon/lint-plugin-evidence",
    namespace: "evidence",
    version,
  } as const,
  rules: ["graph", "singular"] as const,
  source: path.resolve(__dirname, "..", "native"),
} satisfies ITtscLintPlugin;
export default evidence;
