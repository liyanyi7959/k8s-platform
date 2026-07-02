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

const child = spawn(childCommand, childArgs, {
  cwd: path.resolve(__dirname, '..'),
  stdio: ['inherit', 'inherit', 'pipe'],
  shell: process.platform === 'win32',
  env: process.env,
})

const stderr = readline.createInterface({ input: child.stderr })

stderr.on('line', (line) => {
  if (ignoredWatchpackNoise.some((pattern) => pattern.test(line))) {
    return
  }

  process.stderr.write(`${line}\n`)
})

child.on('error', (error) => {
  process.stderr.write(`${error.stack || error.message}\n`)
  process.exit(1)
})

child.on('close', (code, signal) => {
  stderr.close()

  if (signal) {
    process.kill(process.pid, signal)
    return
  }

  process.exit(code ?? 0)
})