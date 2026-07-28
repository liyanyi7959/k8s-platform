const fs = require('node:fs')
const path = require('node:path')

const srcRoot = path.resolve(__dirname, '../src')
const featureRoot = path.join(srcRoot, 'features')
const requiredContexts = ['ai', 'fleet', 'iam', 'incident', 'kops', 'platform', 'provisioning', 'workspace']
const forbiddenDirectories = ['services', 'types']
const forbiddenImports = ['@/services', '@/types']

const walk = (directory) => fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
  const filePath = path.join(directory, entry.name)
  if (entry.isDirectory()) return walk(filePath)
  return /\.(ts|tsx)$/.test(entry.name) ? [filePath] : []
})

const violations = []
for (const context of requiredContexts) {
  if (!fs.existsSync(path.join(featureRoot, context))) {
    violations.push(`missing bounded-context directory: features/${context}`)
  }
}
for (const directory of forbiddenDirectories) {
  if (fs.existsSync(path.join(srcRoot, directory))) {
    violations.push(`legacy horizontal directory must not exist: src/${directory}`)
  }
}
for (const filePath of walk(srcRoot)) {
  const content = fs.readFileSync(filePath, 'utf8')
  for (const importPath of forbiddenImports) {
    if (content.includes(importPath)) {
      violations.push(`${path.relative(srcRoot, filePath)} imports retired alias ${importPath}`)
    }
  }
}

if (violations.length) {
  console.error(violations.join('\n'))
  process.exit(1)
}

console.log('Frontend bounded-context architecture guard passed')
