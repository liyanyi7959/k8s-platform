const fs = require('node:fs')
const path = require('node:path')

const srcRoot = path.resolve(__dirname, '../src')
const featureRoot = path.join(srcRoot, 'features')
const requiredContexts = ['ai', 'fleet', 'iam', 'incident', 'kops', 'platform', 'provisioning', 'workspace']
const forbiddenDirectories = ['schemas', 'services', 'types']
const technicalRoots = ['components', 'hooks', 'shared', 'theme', 'utils']

const walk = (directory) => {
  if (!fs.existsSync(directory)) return []
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const filePath = path.join(directory, entry.name)
    if (entry.isDirectory()) return walk(filePath)
    return /\.(ts|tsx)$/.test(entry.name) ? [filePath] : []
  })
}

const importsOf = (source) => [...source.matchAll(/(?:from\s+|import\s*\()['"]([^'"]+)['"]/g)].map((match) => match[1])
const violations = []

for (const context of requiredContexts) {
  const contextRoot = path.join(featureRoot, context)
  if (!fs.existsSync(contextRoot)) {
    violations.push(`missing bounded-context directory: features/${context}`)
    continue
  }
  if (!fs.existsSync(path.join(contextRoot, 'index.ts'))) {
    violations.push(`missing public feature entry: features/${context}/index.ts`)
  }
}

for (const directory of forbiddenDirectories) {
  if (fs.existsSync(path.join(srcRoot, directory))) {
    violations.push(`legacy horizontal directory must not exist: src/${directory}`)
  }
}

for (const rootName of technicalRoots) {
  for (const filePath of walk(path.join(srcRoot, rootName))) {
    const relativePath = path.relative(srcRoot, filePath)
    for (const importPath of importsOf(fs.readFileSync(filePath, 'utf8'))) {
      if (importPath.startsWith('@/features/')) {
        violations.push(`${relativePath} reverses dependency direction by importing ${importPath}`)
      }
    }
  }
}

for (const owner of requiredContexts) {
  for (const filePath of walk(path.join(featureRoot, owner))) {
    const relativePath = path.relative(srcRoot, filePath)
    for (const importPath of importsOf(fs.readFileSync(filePath, 'utf8'))) {
      const match = importPath.match(/^@\/features\/([^/]+)(\/.*)?$/)
      if (!match || match[1] === owner || !match[2]) continue
      violations.push(`${relativePath} reaches into ${match[1]} internals via ${importPath}; use @/features/${match[1]}`)
    }
  }
}

if (violations.length) {
  console.error(violations.join('\n'))
  process.exit(1)
}

console.log('Frontend bounded-context architecture guard passed')
