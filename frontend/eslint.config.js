import js from '@eslint/js'
import globals from 'globals'
import tseslint from 'typescript-eslint'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'

const BANNED = ['@mui/*', '@emotion/*', 'recharts', 'recharts/*', '@fluentui/*', '@tabler/*', 'axios', 'notistack']

export default tseslint.config(
  { ignores: ['dist', 'node_modules'] },
  {
    files: ['src/**/*.{ts,tsx}', 'e2e/**/*.ts'],
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    languageOptions: { ecmaVersion: 2022, globals: { ...globals.browser, ...globals.node } },
    plugins: { 'react-hooks': reactHooks, 'react-refresh': reactRefresh },
    rules: {
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_', varsIgnorePattern: '^_', ignoreRestSiblings: true }],
      // v7 recommended set, including the React Compiler rules (refs, purity, set-state-in-effect, …).
      ...reactHooks.configs.recommended.rules,
      'react-hooks/exhaustive-deps': 'error',
      'react-refresh/only-export-components': ['warn', { allowConstantExport: true }],
    },
  },
  {
    // Pages seed form state from query data in effects (and a test helper mutates a counter); rewriting
    // them is out of scope for the kit follow-ups, so these two compiler rules stay off for pages only (TODO: re-enable).
    files: ['src/app/pages/**/*.{ts,tsx}'],
    rules: { 'react-hooks/set-state-in-effect': 'off', 'react-hooks/immutability': 'off' },
  },
  {
    // Primitives deliberately co-export small helpers next to their components.
    files: ['src/ui/**', 'src/charts/**', 'src/app/routes.tsx', 'src/app/pages/ItemRedirect.tsx', '**/*.test.{ts,tsx}'],
    rules: { 'react-refresh/only-export-components': 'off' },
  },
  {
    // New code must stay free of the legacy UI stack.
    files: ['src/{styles,lib,ui,charts,app,api}/**/*.{ts,tsx}'],
    rules: {
      'no-restricted-imports': ['error', { patterns: [{ group: BANNED, message: 'Legacy UI stack is not allowed in new code.' }] }],
    },
  },
  {
    // Layering: the ui kit, charts, lib and api client never depend on the app layer.
    files: ['src/{ui,charts,lib,api}/**/*.{ts,tsx}'],
    rules: {
      'no-restricted-imports': ['error', { patterns: [
        { group: BANNED, message: 'Legacy UI stack is not allowed in new code.' },
        { group: ['@/app', '@/app/*'], message: 'ui/charts/lib/api must not import from the app layer.' },
      ] }],
    },
  },
)
