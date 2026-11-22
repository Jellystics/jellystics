import js from '@eslint/js'
import globals from 'globals'
import tseslint from 'typescript-eslint'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'

export default tseslint.config(
  {
    ignores: [
      'dist',
      'node_modules',
      // Legacy code, removed in phase 7.
      'src/features/**',
      'src/shared/**',
      'src/lib/{theme.ts,ThemeModeContext.tsx,PaletteContext.tsx,FaviconContext.tsx,dateRange.tsx,palette.ts,favicon.ts,chartColors.ts,dateLocale.ts,axios.ts,publicApi.ts}',
    ],
  },
  {
    files: ['src/**/*.{ts,tsx}', 'e2e/**/*.ts'],
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    languageOptions: { ecmaVersion: 2022, globals: { ...globals.browser, ...globals.node } },
    plugins: { 'react-hooks': reactHooks, 'react-refresh': reactRefresh },
    rules: {
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_', varsIgnorePattern: '^_', ignoreRestSiblings: true }],
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'error',
      'react-refresh/only-export-components': ['warn', { allowConstantExport: true }],
    },
  },
  {
    // Primitives deliberately co-export small helpers next to their components.
    files: ['src/ui/**', 'src/charts/**', 'src/app/routes.tsx', 'src/app/AppLink.tsx', 'src/app/pages/ItemRedirect.tsx', '**/*.test.{ts,tsx}'],
    rules: { 'react-refresh/only-export-components': 'off' },
  },
)
