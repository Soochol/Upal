import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'
import boundaries from 'eslint-plugin-boundaries'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    plugins: {
      boundaries,
    },
    settings: {
      'boundaries/elements': [
        { type: 'app',      pattern: ['src/app', 'src/app/**'] },
        { type: 'pages',    pattern: ['src/pages', 'src/pages/**'] },
        { type: 'widgets',  pattern: ['src/widgets', 'src/widgets/**'] },
        { type: 'features', pattern: ['src/features', 'src/features/**'] },
        { type: 'entities', pattern: ['src/entities', 'src/entities/**'] },
        { type: 'shared',   pattern: ['src/shared', 'src/shared/**'] },
      ],
    },
    rules: {
      'boundaries/element-types': ['warn', {
        default: 'disallow',
        rules: [
          { from: 'app',      allow: ['pages', 'widgets', 'features', 'entities', 'shared'] },
          { from: 'pages',    allow: ['widgets', 'features', 'entities', 'shared'] },
          { from: 'widgets',  allow: ['features', 'entities', 'shared'] },
          { from: 'features', allow: ['entities', 'shared'] },
          { from: 'entities', allow: ['shared'] },
          { from: 'shared',   allow: [] },
        ],
      }],
    },
  },
])
