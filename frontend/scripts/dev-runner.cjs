const path = require('node:path')
const readline = require('node:readline')
const { spawn } = require('node:child_process')

const maxBin = process.platform === 'win32'
  ? path.resolve(__dirname, '../node_modules/.bin/max.cmd')
  : path.resolve(__dirname, '../node_modules/.bin/max')

const childCommand = maxBin
const childArgs = ['dev', ...process.argv.slice(2)]

const ignoredWatchpackNoise = [
  /Watchpack Error \(initial scan\): Error: EINVAL: invalid argument, lstat 'D:\\DumpStack\.log(?:\.tmp)?'/i,
  /Watchpack Error \(initial scan\): Error: EINVAL: invalid argument, lstat 'D:\\pagefile\.sys'/i,
  /Watchpack Error \(initial scan\): Error: EINVAL: invalid argument, lstat 'D:\\hiberfil\.sys'/i,
  /Watchpack Error \(initial scan\): Error: EINVAL: invalid argument, lstat 'D:\\swapfile\.sys'/i,
  /Watchpack Error \(initial scan\): Error: EINVAL: invalid argument, lstat 'D:\\System Volume Information'/i,
]

// Umi 默认会输出版本、插件及“你知道吗”之类的启动提示。它们不影响开发，
// 但会淹没真正的编译结果；在这里统一过滤，错误和可访问地址仍完整保留。
const umiInformationalNoise = [
  /\[你知道吗？\]/,
  /\[plugin: .*\] \[reactQuery\] use local package, version:/,
  /\bUmi v\d+\.\d+\.\d+$/,
  /^info\s+-\s+Preparing\.\.\.$/,
]

const stripAnsi = (line) => line.replace(/\u001B\[[0-?]*[ -\/]*[@-~]/g, '')

const shouldHideLine = (line) => umiInformationalNoise.some((pattern) => pattern.test(stripAnsi(line)))

const child = spawn(childCommand, childArgs, {
  cwd: path.resolve(__dirname, '..'),
  stdio: ['inherit', 'pipe', 'pipe'],
  shell: process.platform === 'win32',
  env: process.env,
})

const stdout = readline.createInterface({ input: child.stdout })
const stderr = readline.createInterface({ input: child.stderr })

const forwardLine = (line, output) => {
  if (ignoredWatchpackNoise.some((pattern) => pattern.test(line))) {
    return
  }

  if (shouldHideLine(line)) {
    return
  }

  output.write(`${line}\n`)
}

stdout.on('line', (line) => {
  forwardLine(line, process.stdout)
})

stderr.on('line', (line) => {
  forwardLine(line, process.stderr)
})

child.on('error', (error) => {
  process.stderr.write(`${error.stack || error.message}\n`)
  process.exit(1)
})

child.on('close', (code, signal) => {
  stdout.close()
  stderr.close()

  if (signal) {
    process.kill(process.pid, signal)
    return
  }

  process.exit(code ?? 0)
})
