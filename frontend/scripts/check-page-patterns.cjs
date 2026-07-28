const fs = require('node:fs')
const path = require('node:path')

const pagesRoot = path.resolve(__dirname, '../src/features')
const legacyPatterns = [
  'app-server-pool-hero',
  'app-server-pool-summary',
  'app-monitor-fact',
  'app-data-provenance',
  'app-incident-summary',
  'app-data-console__statgrid',
]

const standardListPages = [
  'provisioning/pages/deploy/index.tsx',
  'workspace/pages/projects/index.tsx',
  'iam/pages/system/users.tsx',
  'iam/pages/system/roles.tsx',
  'platform/pages/system/audit-logs.tsx',
]

const standardListComponents = [
  path.resolve(__dirname, '../src/features/kops/components/GenericResourceList/index.tsx'),
]
const listWorkspacePath = path.resolve(__dirname, '../src/components/ListWorkspace/index.tsx')
const formWorkspacePath = path.resolve(__dirname, '../src/components/FormWorkspace/index.tsx')
const globalStylesPath = path.resolve(__dirname, '../src/global.less')

const walk = (directory) => fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
  const filePath = path.join(directory, entry.name)
  if (entry.isDirectory()) return walk(filePath)
  return entry.name.endsWith('.tsx') ? [filePath] : []
})

const violations = []
for (const filePath of walk(pagesRoot)) {
  const content = fs.readFileSync(filePath, 'utf8')
  for (const pattern of legacyPatterns) {
    if (content.includes(pattern)) violations.push(`${path.relative(process.cwd(), filePath)}: ${pattern}`)
  }
}

for (const relativePath of standardListPages) {
  const filePath = path.join(pagesRoot, relativePath)
  const content = fs.readFileSync(filePath, 'utf8')
  if (!content.includes('ListWorkspace')) {
    violations.push(`${path.relative(process.cwd(), filePath)}: management lists must use ListWorkspace`)
  }
  if (!content.includes('summary=')) {
    violations.push(`${path.relative(process.cwd(), filePath)}: management lists must render a live summary row`)
  }
}

for (const filePath of standardListComponents) {
  const content = fs.readFileSync(filePath, 'utf8')
  if (!content.includes('ListWorkspace') || !content.includes('summary=')) {
    violations.push(`${path.relative(process.cwd(), filePath)}: generic resource lists must use ListWorkspace with a live summary`)
  }
}

const listWorkspace = fs.readFileSync(listWorkspacePath, 'utf8')
const formWorkspace = fs.readFileSync(formWorkspacePath, 'utf8')
const globalStyles = fs.readFileSync(globalStylesPath, 'utf8')

if (!listWorkspace.includes('app-list-workspace__content')) {
  violations.push('ListWorkspace: default table content wrapper is missing')
}
if (!globalStyles.includes('--app-list-content-inline: 32px') || !globalStyles.includes('var(--app-list-content-block-start)')) {
  violations.push('ListWorkspace: deployment-list content inset baseline is missing')
}
if (
  !globalStyles.includes('--app-workspace-radius: 16px') ||
  !globalStyles.includes('--app-table-header-bg: #f6f8fc') ||
  !globalStyles.includes('Logged-in workspace foundation')
) {
  violations.push('Workspace foundation: global card radius and table-header color tokens are missing')
}
if (!formWorkspace.includes('app-form-workspace__panel') || !globalStyles.includes('width: min(100%, 960px)')) {
  violations.push('FormWorkspace: readable-width configuration panel baseline is missing')
}

if (violations.length) {
  console.error('Detected retired page-specific visual patterns:')
  console.error(violations.join('\n'))
  process.exit(1)
}

console.log('Page-pattern and list-workspace guard passed')
