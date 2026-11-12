// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors
// The kit's types are written out from the catalogue; they must say what it says. And the
// copies of the catalogue and the schema beside the kit must be WittyModule's own.
import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import { test } from 'node:test'
import * as kit from '../wittymodule-sdui.ts'

const read = (rel) => readFileSync(new URL(rel, import.meta.url), 'utf8')
const catalog = JSON.parse(read('../catalog.json'))
const schema = JSON.parse(read('../sdui-module.schema.json'))
const vals = (o) => o.map((x) => x.value)
const cond = (f) => vals(catalog.conditions.find((c) => c.field === f).options)

test('the type lists are the catalogue\'s', () => {
  assert.equal(kit.CATALOG_REVISION, catalog.revision)
  assert.deepEqual([...kit.ICONS], catalog.icons)
  assert.deepEqual([...kit.CARD_STYLES], catalog.cardStyles.map((c) => c.style))
  assert.deepEqual([...kit.COMPONENTS], catalog.components.filter((c) => c.authored).map((c) => c.type))
  assert.deepEqual([...kit.SOURCES], catalog.sources.map((s) => s.type))
  assert.deepEqual([...kit.PAGE_KINDS], catalog.pageKinds.map((k) => k.kind))
  assert.deepEqual([...kit.FORM_FACTORS], catalog.formFactors)
  assert.deepEqual([...kit.PLATFORMS], catalog.platforms)
  assert.deepEqual([...kit.PROFILE_KINDS], catalog.profileKinds)
  assert.deepEqual([...kit.LIBRARY_KINDS], catalog.libraryKinds)
  assert.deepEqual([...kit.MEDIA_KINDS], catalog.mediaKinds)
  assert.deepEqual([...kit.PLACEMENTS], vals(catalog.placements))
  assert.deepEqual([...kit.REPEATS], vals(catalog.repeats))
  assert.deepEqual([...kit.PERSONAL_KINDS], vals(catalog.personalKinds))
  assert.deepEqual([...kit.NATIVE_VIEWS], vals(catalog.nativeViews))
  assert.deepEqual([...kit.WEEKDAYS], cond('weekday'))
  assert.deepEqual([...kit.CAN_FEATURES], cond('can'))
  assert.deepEqual([...kit.FLAGS], cond('flag'))
  for (const k of kit.MODULE_PAGE_KINDS) assert.ok(kit.PAGE_KINDS.includes(k), k)
  for (const [id, p] of Object.entries(kit.STOCK_PAGES)) assert.ok(kit.PAGE_KINDS.includes(p.kind), id)
})

test('the schema is for the format the kit writes', () => {
  assert.equal(schema['x-format'], kit.FORMAT)
  assert.equal(schema['x-formatVersion'], kit.FORMAT_VERSION)
  assert.equal(schema['x-catalogRevision'], catalog.revision)
})

// In WittyModule the kit's copies sit beside the originals in sdui/; in a module's
// sdui/kit/ (written by module-kit.sh) there is nothing to compare with.
const original = new URL('../../../sdui/catalog.json', import.meta.url)
test('the copies are WittyModule\'s own', { skip: !existsSync(original) && 'not in WittyModule' }, () => {
  assert.equal(read('../catalog.json'), read('../../../sdui/catalog.json'), 'kit/sdui/catalog.json differs from sdui/catalog.json: copy it again')
  assert.equal(read('../sdui-module.schema.json'), read('../../../sdui/sdui-module.schema.json'), 'kit/sdui/sdui-module.schema.json differs from sdui/: copy it again')
})
