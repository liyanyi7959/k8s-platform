const path = require('node:path')
const fs = require('node:fs')
const readline = require('node:readline')
const { spawn } = require('node:child_process')

const maxBin = process.platform === 'win32'
  ? path.resolve(__dirname, '../node_modules/.bin/max.cmd')
  : path.resolve(__dirname, '../node_modules/.bin/max')

const childCommand = maxBin
const childArgs = ['dev', ...process.argv.slice(2)]
const lockPath = path.resolve(__dirname, '../.aiops-dev.lock')

const isProcessRunning = (pid) => {
  if (!Number.isInteger(pid) || pid <= 0) return false
  try {
    process.kill(pid, 0)
    return true
  } catch {
    return false
  }
}

const acquireDevLock = () => {
  for (let attempt = 0; attempt < 2; attempt += 1) {
    try {
      const descriptor = fs.openSync(lockPath, 'wx')
      fs.writeFileSync(descriptor, `${process.pid}\n`)
      fs.closeSync(descriptor)
      return
    } catch (error) {
      if (error.code !== 'EEXIST') throw error
      const ownerPid = Number.parseInt(fs.readFileSync(lockPath, 'utf8'), 10)
      if (isProcessRunning(ownerPid)) {
        throw new Error(`已有前端开发服务正在运行（PID ${ownerPid}）。同一工作区一次只能运行一个 npm run dev；如需并行开发，请使用独立工作区。`)
      }
      fs.rmSync(lockPath, { force: true })
    }
  }
  throw new Error('无法获取前端开发服务锁')
}

const releaseDevLock = () => fs.rmSync(lockPath, { force: true })

try {
  acquireDevLock()
} catch (error) {
  process.stderr.write(`${error.message}\n`)
  process.exit(1)
}

process.once('exit', releaseDevLock)

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
  env: {
    ...process.env,
    // Disable Umi's startup promotional tips at the source.
    DID_YOU_KNOW: process.env.DID_YOU_KNOW || 'none',
  },
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
