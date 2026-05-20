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

const config = [
  { ignores },
  ...nextVitals,
  ...nextTypescript,
  {
    rules: {
      "@typescript-eslint/no-explicit-any": "off",
      "react-hooks/immutability": "off",
      "react-hooks/purity": "off",
      "react-hooks/set-state-in-effect": "off",
      "react-hooks/static-components": "off",
    },
  },
]

export default config
