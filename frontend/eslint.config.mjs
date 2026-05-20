import nextVitals from "eslint-config-next/core-web-vitals"
import nextTypescript from "eslint-config-next/typescript"

const ignores = [
  ".next/**",
  "out/**",
  "build/**",
  "dist/**",
  "next-env.d.ts",
  "tsconfig.tsbuildinfo",
]

export default [
  { ignores },
  ...nextVitals,
  ...nextTypescript,
  {
    rules: {
      "@typescript-eslint/no-explicit-any": "off",
    },
  },
]
