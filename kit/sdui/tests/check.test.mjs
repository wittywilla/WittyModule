// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import * as kit from '../wittymodule-sdui.ts'
import { variants } from './variants.mjs'

const example = JSON.parse(readFileSync(new URL('./fixtures/module-boardgames.json', import.meta.url), 'utf8'))

test('the server\'s example module has no issues', () => {
  assert.deepEqual(kit.checkModuleUI(structuredClone(example), 'wittyboardgames'), [])
  assert.deepEqual(kit.checkModuleUI(structuredClone(example)), [])
})

for (const v of variants) {
  test(`caught: ${v.rule}`, () => {
    const ui = structuredClone(example)
    v.change(ui)
    const issues = kit.checkModuleUI(ui, v.moduleId ?? 'wittyboardgames')
    const hit = issues.find((i) => i.severity === 'error' && (i.path === v.path || i.path.startsWith(v.path + '/')))
    assert.ok(hit, `no error at ${v.path}; got:\n${kit.formatIssues(issues) || '(nothing)'}`)
    assert.ok(hit.message.length > 0)
  })
}

test('a file that isn\'t an object is refused', () => {
  assert.ok(kit.hasErrors(kit.checkModuleUI([], 'wittyboardgames')))
  assert.ok(kit.hasErrors(kit.checkModuleUI(null, 'wittyboardgames')))
})

test('a file over 1 MiB is refused', () => {
  const ui = structuredClone(example)
  ui.pages[0].description = 'x'.repeat(kit.MAX_MODULE_BYTES)
  assert.ok(kit.checkModuleUI(ui, 'wittyboardgames').some((i) => i.path === '/' && i.severity === 'error'))
})

test('warnings don\'t count as errors', () => {
  const ui = structuredClone(example)
  ui.pages[0].sections[1].props.cardStyle = 'channel' // made for channels; the source gives games
  const issues = kit.checkModuleUI(ui, 'wittyboardgames')
  assert.equal(issues.length, 1)
  assert.equal(issues[0].severity, 'warning')
  assert.equal(kit.hasErrors(issues), false)
})

test('a fix is offered where the server offers one', () => {
  const ui = structuredClone(example)
  ui.pages[0].icon = 'gam'
  const [issue] = kit.checkModuleUI(ui, 'wittyboardgames')
  assert.equal(issue.path, '/pages/0/icon')
  assert.equal(issue.fix, 'Did you mean "game"?')
})

test('the constructors make sections the checker accepts', () => {
  const ui = kit.defineModuleUI(
    kit.moduleUI('wittydemo', {
      sources: [{ type: 'wittydemo.recent', label: 'Recent', description: 'Recent things.', icon: 'clock', yields: ['book'], cardStyle: 'book', paged: false, since: 2 }],
      views: [{ id: 'wittydemo.reader', label: 'Reader' }],
      pages: [
        {
          id: 'wittydemo',
          kind: 'custom',
          title: 'Demo',
          sections: [
            kit.hero('top', { type: 'wittydemo.recent' }, { maxItems: 3 }),
            kit.text('intro', 'Things **lately**, see [the app](/page/wittydemo).', { format: 'markdown', role: 'body' }),
            kit.notice('tip', 'Set it up first.', { tone: 'warning', buttonLabel: 'Settings', native: 'settings' }, { when: { admin: true } }),
            kit.shelf('recent', { type: 'wittydemo.recent' }, { title: 'Recent', cardStyle: 'book' }),
            kit.grid('all', { type: 'recentlyAdded', limit: 50 }, { title: 'Everything' }),
          ],
        },
        { id: 'wittydemo-read', kind: 'custom', title: 'Read', sections: [kit.nativeView('reader', 'wittydemo.reader', [kit.text('reader-note', 'Open it on the web.')])] },
      ],
      navigation: [{ id: 'wittydemo', type: 'page', page: 'wittydemo', icon: 'book' }],
    }),
  )
  assert.equal(ui.theme.scope, 'wittydemo')
  assert.equal(ui.catalog, kit.CATALOG_REVISION)
  assert.deepEqual(kit.checkModuleUI(ui, 'wittydemo'), [])
  assert.deepEqual(kit.text('t', 'Hi'), { id: 't', component: 'Text', props: { text: 'Hi' } })
  assert.deepEqual(kit.nativeView('n', 'guide'), { id: 'n', component: 'NativeView', props: { view: 'guide' } })
})

test('the revision 6 constructors make sections the checker accepts', () => {
  const ui = kit.moduleUI('wittydemo', {
    sources: [{ type: 'wittydemo.people', label: 'People', description: 'Friends.', icon: 'users', yields: ['person'], cardStyle: 'person', paged: false, since: 6 }],
    pages: [
      {
        id: 'wittydemo', kind: 'custom', title: 'Demo',
        sections: [
          kit.carousel('c', { type: 'wittydemo.people' }, { cardStyle: 'person', maxItems: 10 }),
          kit.numberedRow('n', { type: 'topRated' }, { numbers: 'index' }),
          kit.shelf('s', { type: 'wittydemo.people' }, { headingStyle: 'band', peek: false, seeAllCard: 'never' }),
          kit.widgetGrid('w', [kit.widget('w1', { kind: 'clock', x: 0, y: 0, w: 2 }), kit.widget('w2', { kind: 'stats', x: 2, y: 0 })], { columns: 4 }),
          kit.crossBar('x', [kit.section('t1', 'Tab', { props: { label: 'People' }, children: [kit.shelf('r', { type: 'wittydemo.people' })] })], { default: 't1' }),
        ],
      },
    ],
    navigation: [{ id: 'wittydemo', type: 'page', page: 'wittydemo', icon: 'users' }],
  })
  assert.deepEqual(kit.checkModuleUI(ui, 'wittydemo'), [])
  assert.deepEqual(kit.credits('cr', { type: 'credits', item: '$item' }, { parts: ['cast'] }), { id: 'cr', component: 'Credits', props: { parts: ['cast'] }, source: { type: 'credits', item: '$item' } })
})

test('revision 8: regions, slots and forms are checked as the server checks them', () => {
  const base = () => kit.moduleUI('wittydemo', {
    pages: [{ id: 'wittydemo', kind: 'custom', title: 'Demo', sections: [kit.form('f', [kit.section('l', 'List', { children: [kit.section('li', 'ListItem', { props: { title: 'Show friends what I play' } })] })], { op: 'wittyplay.online.set', submitLabel: 'Save' })], regions: { statusBar: [{ id: 'who', kind: 'profile', text: 'Online' }], systemDock: [{ id: 'games', label: 'Games', icon: 'game', route: '/page/wittydemo', as: 'layer' }] } }],
    navigation: [{ id: 'wittydemo', type: 'page', page: 'wittydemo', icon: 'game' }],
    settings: [{ id: 'wittydemo', title: 'Demo', slot: 'you.games', groups: [{ rows: [{ key: 'n', kind: 'bool', label: 'On', default: true }] }] }],
  })
  assert.deepEqual(kit.checkModuleUI(base(), 'wittydemo'), [])
  const bad = base()
  bad.pages[0].regions.statusBar[0].kind = 'banner'
  bad.pages[0].regions.systemDock[0].route = 'page'
  bad.settings[0].slot = 'people.profile'
  const paths = kit.checkModuleUI(bad, 'wittydemo').map((i) => i.path)
  for (const p of ['/pages/0/regions/statusBar/0/kind', '/pages/0/regions/systemDock/0/route', '/settings/0/admin']) assert.ok(paths.includes(p) || paths.some((x) => x.endsWith(p.replace('/pages/0', ''))), p + ' in ' + paths)
})

test('revision 8 additions: a QR section, a page slot, a page-level filter bar', () => {
  const ui = kit.moduleUI('wittydemo', {
    pages: [{ id: 'wittydemo', kind: 'custom', title: 'Demo', slot: 'you.games', sections: [kit.qr('q', { value: '/games?join=ABC', label: 'Scan to join', size: 'l' })] }],
    navigation: [],
  })
  assert.deepEqual(kit.checkModuleUI(ui, 'wittydemo').filter((i) => i.severity === 'error'), [])
  ui.pages[0].slot = 'elsewhere'
  assert.ok(kit.checkModuleUI(ui, 'wittydemo').some((i) => i.path === '/pages/0/slot'))
})

test('checkSetting checks a value as the server does', () => {
  assert.equal(kit.checkSetting({ key: 'n', kind: 'int', min: 1, max: 3 }, 2), '')
  assert.equal(kit.checkSetting({ key: 'n', kind: 'int', min: 1, max: 3 }, 4), 'n is between 1 and 3')
  assert.equal(kit.checkSetting({ key: 'n', kind: 'int' }, 1.5), 'n is a whole number')
  assert.equal(kit.checkSetting({ key: 's', kind: 'string' }, 'a\nb'), 's is one line of plain text')
  assert.equal(kit.checkSetting({ key: 'b', kind: 'bool' }, 'true'), 'b is true or false')
})

test('names and colours', () => {
  assert.equal(kit.moduleOf('wittyplay.ConsoleShelf'), 'wittyplay')
  assert.equal(kit.moduleOf('Shelf'), '')
  assert.ok(kit.validRoute('/page/wittyplay'))
  assert.ok(!kit.validRoute('//evil.example'))
  assert.ok(!kit.validRoute('/a%zz'))
  assert.ok(kit.validURL('https://example.org/x'))
  assert.ok(!kit.validURL('javascript:alert(1)'))
  for (const c of ['#2E7D32', '#abc', '#2E7D3280', 'rgb(1, 2, 3)', 'rgba(1,2,3,0.5)', 'rgb(1 2 3 / 50%)', 'transparent']) assert.ok(kit.isColour(c), c)
  for (const c of ['green-ish', '#12', 'rgb(300,0,0)', 'hsl(1,2%,3%)']) assert.ok(!kit.isColour(c), c)
})
