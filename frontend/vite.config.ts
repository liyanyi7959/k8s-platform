import { fileURLToPath, URL } from 'node:url'
import fs from 'node:fs'
import path from 'node:path'
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

function editormdAssets() {
  const mountPath = '/vendor/editormd'
  const srcRoot = path.resolve(process.cwd(), 'node_modules/editor.md')
  const allowList = new Set(['css', 'lib', 'plugins', 'fonts', 'images', 'languages', 'editormd.min.js', 'editormd.js'])

  function isAllowed(relPath: string) {
    const seg = relPath.split('/').filter(Boolean)[0] ?? ''
    return allowList.has(seg)
  }

  function mimeTypeOf(filePath: string) {
    const ext = path.extname(filePath).toLowerCase()
    if (ext === '.js') return 'text/javascript'
    if (ext === '.css') return 'text/css'
    if (ext === '.json') return 'application/json'
    if (ext === '.png') return 'image/png'
    if (ext === '.gif') return 'image/gif'
    if (ext === '.jpg' || ext === '.jpeg') return 'image/jpeg'
    if (ext === '.svg') return 'image/svg+xml'
    if (ext === '.woff') return 'font/woff'
    if (ext === '.woff2') return 'font/woff2'
    if (ext === '.ttf') return 'font/ttf'
    if (ext === '.eot') return 'application/vnd.ms-fontobject'
    if (ext === '.otf') return 'font/otf'
    return 'application/octet-stream'
  }

  function copyFileSync(src: string, dest: string) {
    fs.mkdirSync(path.dirname(dest), { recursive: true })
    fs.copyFileSync(src, dest)
  }

  function copyDirSync(srcDir: string, destDir: string) {
    fs.mkdirSync(destDir, { recursive: true })
    for (const ent of fs.readdirSync(srcDir, { withFileTypes: true })) {
      const src = path.join(srcDir, ent.name)
      const dest = path.join(destDir, ent.name)
      if (ent.isDirectory()) {
        copyDirSync(src, dest)
      } else if (ent.isFile()) {
        copyFileSync(src, dest)
      }
    }
  }

  return {
    name: 'k8s-platform:editormd-assets',
    configureServer(server: any) {
      server.middlewares.use(mountPath, (req: any, res: any, next: any) => {
        const urlPath = decodeURIComponent(String(req.url || '/').split('?')[0] || '/')
        const rel = urlPath.replace(/^\/+/, '')
        if (!isAllowed(rel)) return next()

        const filePath = path.join(srcRoot, rel)
        if (!filePath.startsWith(srcRoot)) return next()
        if (!fs.existsSync(filePath) || fs.statSync(filePath).isDirectory()) return next()

        res.statusCode = 200
        res.setHeader('Content-Type', mimeTypeOf(filePath))
        fs.createReadStream(filePath).pipe(res)
      })
    },
    writeBundle(outputOptions: any) {
      const outDir = outputOptions.dir ?? (outputOptions.file ? path.dirname(outputOptions.file) : null)
      if (!outDir) return
      const destRoot = path.join(outDir, mountPath.replace(/^\//, ''))

      for (const ent of allowList) {
        const src = path.join(srcRoot, ent)
        const dest = path.join(destRoot, ent)
        if (!fs.existsSync(src)) continue
        if (fs.statSync(src).isDirectory()) copyDirSync(src, dest)
        else copyFileSync(src, dest)
      }
    }
  }
}

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const proxyTarget = env.VITE_PROXY_TARGET || 'http://localhost:8080'

  function attachProxyErrorFilter(proxy: any) {
    proxy.on('error', (err: NodeJS.ErrnoException, _req: any, socket: any) => {
      if (err?.code === 'ECONNRESET' || err?.code === 'EPIPE') {
        if (socket && typeof socket.destroy === 'function' && !socket.destroyed) {
          socket.destroy()
        }
        return
      }
    })
  }

  return {
    plugins: [vue(), editormdAssets()],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url))
      }
    },
    server: {
      port: 5173,
      strictPort: true,
      proxy: {
        '/api': {
          target: proxyTarget,
          changeOrigin: false,
          ws: true,
          configure: attachProxyErrorFilter
        },
        '/uploads': {
          target: proxyTarget,
          changeOrigin: false
        }
      }
    }
  }
})
