module.exports = {
  root: true,
  env: {
    browser: true,
    es2022: true
  },
  overrides: [
    {
      files: ['*.cjs'],
      env: { node: true, browser: false },
      parserOptions: { sourceType: 'script' }
    }
  ],
  parser: 'vue-eslint-parser',
  parserOptions: {
    parser: '@typescript-eslint/parser',
    ecmaVersion: 'latest',
    sourceType: 'module'
  },
  plugins: ['vue', '@typescript-eslint'],
  extends: ['eslint:recommended', 'plugin:vue/vue3-recommended', 'plugin:@typescript-eslint/recommended'],
  rules: {
    '@typescript-eslint/no-explicit-any': 'off',
    '@typescript-eslint/no-unused-vars': ['error', { "argsIgnorePattern": "^_", "varsIgnorePattern": "^_" }],
    'vue/multi-word-component-names': 'off',
    'vue/no-mutating-props': 'off',
    'vue/max-attributes-per-line': 'off',
    'vue/singleline-html-element-content-newline': 'off',
    'vue/multiline-html-element-content-newline': 'off',
    'vue/html-indent': 'off',
    'vue/html-closing-bracket-newline': 'off',
    'vue/first-attribute-linebreak': 'off',
    'vue/attributes-order': 'off'
  }
}
