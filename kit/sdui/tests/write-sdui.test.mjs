// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors
import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import { mkdtempSync, readFileSync, rmSync, writeFileSync, existsSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { after, test } from 'node:test'

const script = fileURLToPath(new URL('../write-sdui.mjs', import.meta.url))
const fixture = (f) => fileURLToPath(new URL('./fixtures/' + f, import.meta.url))
const dir = mkdtempSync(join(tmpdir(), 'write-sdui-'))
after(() => rmSync(dir, { recursive: true, force: true }))
const run = (...args) => spawnSync(process.execPath, [script, ...args], { encoding: 'utf8' })

test('writes sdui.json from a TypeScript definition, the same as the server\'s example', () => {
  const out = join(dir, 'sdui.json')
  const r = run('--module', 'wittyboardgames', fixture('boardgames-sdui.ts'), out)
  assert.equal(r.status, 0, r.stderr)
  // The kit writes the revision it knows; the server's example was made for an earlier one,
  // which the server still takes. Everything else is the same.
  const got = JSON.parse(readFileSync(out, 'utf8')), want = JSON.parse(readFileSync(fixture('module-boardgames.json'), 'utf8'))
  assert.equal(got.catalog, JSON.parse(readFileSync(new URL('../catalog.json', import.meta.url), 'utf8')).revision)
  assert.deepEqual({ ...got, catalog: want.catalog }, want)
})

test('takes the module ID from module.json beside the output', () => {
  writeFileSync(join(dir, 'module.json'), JSON.stringify({ id: 'wittychess' }))
  const out = join(dir, 'sdui.json')
  rmSync(out, { force: true })
  const r = run(fixture('boardgames-sdui.ts'), out)
  assert.equal(r.status, 1)
  assert.match(r.stderr, /\/module: the file is for "wittyboardgames", and the module is "wittychess"/)
  assert.ok(!existsSync(out), 'nothing is written on an error')
  rmSync(join(dir, 'module.json'))
})

test('fails on errors and writes nothing', () => {
  const broken = JSON.parse(readFileSync(fixture('module-boardgames.json'), 'utf8'))
  broken.pages[0].sections[0].component = 'wittyplay.ConsoleShelf'
  const src = join(dir, 'broken.json')
  writeFileSync(src, JSON.stringify(broken))
  const out = join(dir, 'broken-out.json')
  const r = run('--module', 'wittyboardgames', src, out)
  assert.equal(r.status, 1)
  assert.match(r.stderr, /error \/pages\/0\/sections\/0\/component: "wittyplay.ConsoleShelf" is another module's component/)
  assert.ok(!existsSync(out))
})

test('--check checks a file written by hand', () => {
  assert.equal(run('--module', 'wittyboardgames', '--check', fixture('module-boardgames.json')).status, 0)
  const withSchema = JSON.parse(readFileSync(fixture('module-boardgames.json'), 'utf8'))
  withSchema.$schema = './sdui/kit/sdui-module.schema.json'
  const f = join(dir, 'hand.json')
  writeFileSync(f, JSON.stringify(withSchema))
  assert.equal(run('--check', f).status, 0, 'a $schema naming the schema is allowed')
  withSchema.extra = true
  writeFileSync(f, JSON.stringify(withSchema))
  const r = run('--check', f)
  assert.equal(r.status, 1)
  assert.match(r.stderr, /\/extra/)
})

test('a bad command line is explained', () => {
  const r = run('only-one-file.ts')
  assert.equal(r.status, 2)
  assert.match(r.stderr, /usage:/)
})
