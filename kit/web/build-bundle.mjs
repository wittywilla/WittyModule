// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors
// Builds a module's part in the browser as a web bundle a server can load without being
// updated: one script (index.js) and one stylesheet (style.css), for `wittymodule pack`.
//
//   node <WittyModule>/kit/web/build-bundle.mjs <module folder> [output folder]
//   npm run build:bundle                       (in a module's folder, using its kit/web copy)
//
// This is WittyModule's kit/web/build-bundle.mjs: the one builder every module's web part
// is built with (`node <WittyModule>/kit/web/build-bundle.mjs .`, or a copy kit/module-kit.sh
// writes into the module's kit/web/). Web host API 5 shares the app's router too.
//
// What the app already has is not bundled again. React has to be the app's own, or
// hooks break, and every reader has to share one WittyInk, or the host they are given
// can't be found. So those imports become small stand-ins that read the app's copies
// from globalThis.__wmsModuleHost (web/src/moduleHost.ts). The page's security policy
// has no room for an inline import map, and this needs none. The names are listed here
// at build time because a script's named imports are fixed before it runs.
import { existsSync, readFileSync, statSync } from 'node:fs'
import { createRequire } from 'node:module'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { pathToFileURL } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))

function moduleFolder(arg) {
  const candidates = arg ? [resolve(arg), resolve(here, '../src/modules', arg)] : [here, process.cwd()]
  const dir = candidates.find((d) => existsSync(join(d, 'module.json')) && statSync(d).isDirectory())
  if (!dir) throw new Error(arg ? `no module named ${arg} (no module.json there)` : 'run this in a module folder, or name one')
  return dir
}

const dir = moduleFolder(process.argv[2])
const out = resolve(process.argv[3] || join(dir, 'dist'))
const manifest = JSON.parse(readFileSync(join(dir, 'module.json'), 'utf8'))
const require = createRequire(join(dir, 'module.json'))
// The module's own Vite, wherever this builder is.
const { build, version: viteVersion } = await import(pathToFileURL(require.resolve('vite')).href)
process.env.NODE_ENV = 'production'

const entry = ['index.ts', 'index.tsx', 'index.js', 'web/index.ts', 'web/index.tsx'].map((f) => join(dir, f)).find(existsSync)
if (!entry) throw new Error(`${dir} has no index.ts`)

// WittyInk sits beside the readers in the app, and in node_modules in a reader's own
// repository.
function inkEntry() {
  const beside = join(dir, '../wittyink/index.ts')
  return existsSync(beside) ? beside : require.resolve('@wittywilla/wittyink')
}

// What an ES module or TypeScript file exports, by asking the bundler itself.
async function esExports(file) {
  const res = await build({
    configFile: false, logLevel: 'silent', publicDir: false,
    esbuild: { jsx: 'automatic', jsxDev: false },
    build: { write: false, minify: false, cssCodeSplit: false, rollupOptions: { input: file, preserveEntrySignatures: 'strict', external: (id) => /^react(-dom)?(\/|$)/.test(id) } },
  })
  const chunk = [res].flat().flatMap((r) => r.output).find((o) => o.type === 'chunk' && o.isEntry)
  return chunk.exports
}

const cjsExports = (name) => Object.keys(require(name))

// The stand-in for one shared library. `from` is the expression holding the app's copy.
function standIn(from, names) {
  const lines = [`const m = ${from};`, 'export default m;']
  for (const n of new Set(names)) {
    if (n !== 'default' && n !== '__esModule' && /^[A-Za-z_$][\w$]*$/.test(n)) lines.push(`export const ${n} = m.${n};`)
  }
  return lines.join('\n')
}

// One script, whichever bundler: Vite 8's Rolldown calls it codeSplitting, and warns at the old name.
const oneFile = Number(viteVersion.split('.')[0]) >= 8 ? { codeSplitting: false } : { inlineDynamicImports: true }

const HOST = 'globalThis.__wmsModuleHost'
const shared = {
  react: async () => standIn(`${HOST}.react`, cjsExports('react')),
  'react/jsx-runtime': async () => standIn(`${HOST}.jsxRuntime`, cjsExports('react/jsx-runtime')),
  'react-dom': async () => standIn(`${HOST}.reactDom`, cjsExports('react-dom')),
  // The app's router (web host API 5): a module's routes and links are the app's, which a
  // second copy's context would never see.
  'react-router-dom': async () => standIn(`${HOST}.reactRouterDom`, await esExports(require.resolve('react-router-dom'))),
  'react-router': async () => standIn(`${HOST}.reactRouterDom`, await esExports(require.resolve('react-router-dom'))),
  '@wittywilla/wittyink': async () => standIn(`${HOST}.wittyink`, await esExports(inkEntry())),
  // The loader has pdf.js in hand before it imports a bundle that needs it, so no
  // top-level await is needed and older browsers can still run the bundle.
  'pdfjs-dist': async () => standIn(`${HOST}.pdfjsModule`, await esExports(require.resolve('pdfjs-dist/legacy/build/pdf.mjs'))),
  // A worker's own address decides what it may do. The app's copy sits under the
  // page's policy; one served from a bundle would get the API's, which forbids it.
  'pdfjs-dist/worker': async () => `export default ${HOST}.pdfWorkerUrl;`,
}

// The app's own parts (web host API 5): @wms/app/<path> is the app's web/src/<path>, read from
// HOST.app, with the exports kit/web/app-exports.json lists (written by the server's
// scripts/export-module-kit.mjs). A part that isn't listed can't be used.
const appExports = JSON.parse(readFileSync(join(here, 'app-exports.json'), 'utf8')).app
for (const [path, names] of Object.entries(appExports)) {
  // An ES module's default export is its own `default`, not the module (unlike React's CommonJS).
  shared['@wms/app/' + path] = async () => {
    const from = `${HOST}.app[${JSON.stringify(path)}]`
    const lines = [`const m = ${from};`]
    if (names.includes('default')) lines.push('export default m.default;')
    for (const n of names) if (n !== 'default' && /^[A-Za-z_$][\w$]*$/.test(n)) lines.push(`export const ${n} = m.${n};`)
    return lines.join('\n')
  }
}

function sharedName(source) {
  if (source in shared) return source
  if (source.startsWith('@wms/app/')) throw new Error(`${source}: the app doesn't share that part (kit/web/app-exports.json)`)
  if (/^pdfjs-dist\/.*worker.*\?url$/.test(source)) return 'pdfjs-dist/worker'
  if (/^pdfjs-dist\/(legacy\/)?build\/pdf(\.min)?\.mjs$/.test(source)) return 'pdfjs-dist'
  return null
}

const VIRTUAL = '\0wms-host:'
const fromApp = {
  name: 'wms-module-host',
  enforce: 'pre',
  resolveId(source) {
    const name = sharedName(source)
    if (name) return VIRTUAL + name
    // Anything else from these would be a second copy of what must be the app's.
    if (/^(react|react-dom|react-router|react-router-dom|pdfjs-dist)(\/|$)/.test(source)) this.error(`a bundle can't import ${source}: the app doesn't share it (WittyMediaServer web/src/moduleHost.ts)`)
    return null
  },
  load(id) {
    return id.startsWith(VIRTUAL) ? shared[id.slice(VIRTUAL.length)]() : null
  },
}

await build({
  configFile: false, root: dir, base: './', mode: 'production', publicDir: false, logLevel: 'warn',
  plugins: [fromApp],
  esbuild: { jsx: 'automatic', jsxDev: false, legalComments: 'none' },
  define: { 'process.env.NODE_ENV': '"production"' },
  build: {
    outDir: out, emptyOutDir: true, sourcemap: false, cssCodeSplit: false, modulePreload: false, assetsInlineLimit: 1 << 20,
    rollupOptions: {
      input: entry,
      preserveEntrySignatures: 'strict',
      // Reading a shared library's export has no effect, so unused ones can go.
      treeshake: { propertyReadSideEffects: false },
      output: { format: 'es', ...oneFile, entryFileNames: 'index.js', assetFileNames: (a) => (a.names?.some((n) => n.endsWith('.css')) ? 'style.css' : '[name]-[hash][extname]') },
    },
  },
})
console.log(`built ${manifest.id} ${manifest.version} in ${out}`)
