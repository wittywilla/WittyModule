// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

// Broken variants of the server's example module, one per rule. Each changes a copy of
// the example and names the path of the error the server reports for it (the start of
// the path is enough). check.test.mjs runs them through checkModuleUI.

const shelfOf = (id, extra = {}) => ({ id, component: 'Shelf', source: { type: 'wittyboardgames.shelf' }, ...extra })

export const variants = [
  // The file itself.
  { rule: 'format', path: '/format', change: (ui) => { ui.format = 'wittymodule-ui' } },
  { rule: 'format version', path: '/formatVersion', change: (ui) => { ui.formatVersion = 2 } },
  { rule: 'module ID', path: '/module', change: (ui) => { ui.module = 'Witty-Board' } },
  { rule: 'module is module.json\'s', path: '/module', moduleId: 'wittychess', change: () => {} },
  { rule: 'catalogue revision', path: '/catalog', change: (ui) => { ui.catalog = 99 } },
  { rule: 'unknown field', path: '/scheme', change: (ui) => { ui.scheme = './sdui/kit/sdui-module.schema.json' } },
  { rule: 'unknown nested field', path: '/pages/0/sections/1/prop', change: (ui) => { ui.pages[0].sections[1].prop = {} } },
  { rule: 'wrong JSON type', path: '/formatVersion', change: (ui) => { ui.formatVersion = '1' } },
  // The theme scope.
  { rule: 'theme scope is the module ID', path: '/theme/scope', change: (ui) => { ui.theme.scope = 'boardgames' } },
  { rule: 'colour token', path: '/theme/tokens/accentt', change: (ui) => { ui.theme.tokens = { accentt: '#2E7D32' } } },
  { rule: 'colour value', path: '/theme/tokens/accent', change: (ui) => { ui.theme.tokens.accent = 'green-ish' } },
  // Limits.
  { rule: 'at most 32 pages', path: '/pages', change: (ui) => { for (let i = 0; i < 32; i++) ui.pages.push({ ...structuredClone(ui.pages[0]), id: 'wittyboardgames-p' + i }) } },
  { rule: 'at most 16 navigation entries', path: '/navigation', change: (ui) => { for (let i = 0; i < 16; i++) ui.navigation.push({ ...ui.navigation[0], id: 'wittyboardgames-n' + i }) } },
  { rule: 'at most 64 rows on a settings page', path: '/settings/0/groups', change: (ui) => { for (let i = 0; i < 64; i++) ui.settings[0].groups[0].rows.push({ key: 'k' + i, kind: 'bool', label: 'Row ' + i }) } },
  // Sources.
  { rule: 'source is <module>.<camelName>', path: '/sources/0/type', change: (ui) => rename(ui, 'wittyboardgames.shelf', 'wittyboardgames.Shelf') },
  { rule: 'source is the module\'s', path: '/sources/0/type', change: (ui) => { ui.sources[0].type = 'wittyplay.shelf' } },
  { rule: 'source names are unique', path: '/sources/1/type', change: (ui) => { ui.sources.push(structuredClone(ui.sources[0])) } },
  { rule: 'source needs a label', path: '/sources/0/label', change: (ui) => { ui.sources[0].label = ' ' } },
  { rule: 'source icon is the catalogue\'s', path: '/sources/0/icon', change: (ui) => { ui.sources[0].icon = 'dice' } },
  { rule: 'source yields', path: '/sources/0/yields', change: (ui) => { ui.sources[0].yields = [] } },
  { rule: 'source yields an item kind', path: '/sources/0/yields/0', change: (ui) => { ui.sources[0].yields = ['boardgame'] } },
  { rule: 'source card style is the catalogue\'s', path: '/sources/0/cardStyle', change: (ui) => { ui.sources[0].cardStyle = 'tile-xl' } },
  { rule: 'source page kinds are the catalogue\'s', path: '/sources/0/pageKinds/0', change: (ui) => { ui.sources[0].pageKinds = ['boardgames'] } },
  { rule: 'source seeAll is a route', path: '/sources/0/seeAll', change: (ui) => { ui.sources[0].seeAll = 'https://example.org/' } },
  { rule: 'param name', path: '/sources/0/params/0/name', change: (ui) => { ui.sources[0].params[0].name = 'map' } },
  { rule: 'param type', path: '/sources/0/params/0/type', change: (ui) => { ui.sources[0].params[0].type = 'number' } },
  { rule: 'param enum has options', path: '/sources/0/params/1/options', change: (ui) => { ui.sources[0].params[1].options = [] } },
  { rule: 'param default is a value of it', path: '/sources/0/params/0/default', change: (ui) => { ui.sources[0].params[0].default = 40 } },
  // Components.
  { rule: 'component is <module>.<PascalName>', path: '/components/0/type', change: (ui) => rename(ui, 'wittyboardgames.GameNight', 'wittyboardgames.gameNight') },
  { rule: 'component is the module\'s', path: '/components/0/type', change: (ui) => { ui.components[0].type = 'wittyplay.GameNight' } },
  { rule: 'component icon', path: '/components/0/icon', change: (ui) => { ui.components[0].icon = 'meeple' } },
  { rule: 'component page kinds', path: '/components/0/pageKinds/0', change: (ui) => { ui.components[0].pageKinds = ['boardgames'] } },
  { rule: 'placeholder names a property', path: '/components/0/section', change: (ui) => { ui.components[0].section.props.subtitle = 'In {{colour}}' } },
  { rule: 'component made of catalogue components', path: '/components/0/section/component', change: (ui) => { ui.components.push({ type: 'wittyboardgames.Twice', label: 'Twice', section: { id: 'x', component: 'wittyboardgames.GameNight' } }); ui.components.reverse() } },
  { rule: 'component template is a valid section', path: '/components/0/section/0/props/cardSize', change: (ui) => { ui.components[0].section.props.cardSize = 'xl' } },
  { rule: 'component template uses catalogue components', path: '/components/0/section/0/component', change: (ui) => { ui.components[0].section.component = 'Carousel' } },
  // Views.
  { rule: 'view is <module>.<camelName>', path: '/views/0/id', change: (ui) => { ui.views[0].id = 'scoreboard'; ui.pages[0].sections[2].props.view = 'scoreboard' } },
  { rule: 'view platform', path: '/views/0/platforms/0', change: (ui) => { ui.views[0].platforms = ['webos'] } },
  // Pages.
  { rule: 'page ID is the module\'s', path: '/pages/0/id', change: (ui) => { ui.pages[0].id = 'boardgames'; ui.navigation[0].page = 'boardgames' } },
  { rule: 'page kind a module may add', path: '/pages/0/kind', change: (ui) => { ui.pages[0].kind = 'home' } },
  { rule: 'page kind is a kind', path: '/pages/0/kind', change: (ui) => { ui.pages[0].kind = 'settings' } },
  { rule: 'library page is matched', path: '/pages/0/match', change: (ui) => { ui.pages[0].kind = 'library' } },
  { rule: 'no library IDs', path: '/pages/0/match/library', change: (ui) => { ui.pages[0].kind = 'library'; ui.pages[0].match = { libraryKind: 'games', library: 3 } } },
  { rule: 'page icon', path: '/pages/0/icon', change: (ui) => { ui.pages[0].icon = 'dice' } },
  { rule: 'page title', path: '/pages/0/title', change: (ui) => { ui.pages[0].title = '' } },
  { rule: 'section IDs', path: '/pages/0/sections/1/id', change: (ui) => { ui.pages[0].sections[1].id = 'Every Game' } },
  { rule: 'section IDs are unique', path: '/pages/0/sections/1/id', change: (ui) => { ui.pages[0].sections[1].id = 'night' } },
  { rule: 'another module\'s component', path: '/pages/0/sections/0/component', change: (ui) => { ui.pages[0].sections[0].component = 'wittyplay.ConsoleShelf' } },
  { rule: 'another module\'s source', path: '/pages/0/sections/1/source/type', change: (ui) => { ui.pages[0].sections[1].source.type = 'wittyplay.recent' } },
  { rule: 'catalogue component', path: '/pages/0/sections/1/component', change: (ui) => { ui.pages[0].sections[1].component = 'Carousel' } },
  { rule: 'catalogue source', path: '/pages/0/sections/1/source/type', change: (ui) => { ui.pages[0].sections[1].source = { type: 'recentlyAddedd' } } },
  { rule: 'component props', path: '/pages/0/sections/1/props/cardStyle', change: (ui) => { ui.pages[0].sections[1].props.cardStyle = 'giant' } },
  { rule: 'component has the prop', path: '/pages/0/sections/1/props/colour', change: (ui) => { ui.pages[0].sections[1].props.colour = 'red' } },
  { rule: 'source params', path: '/pages/0/sections/1/source/sort', change: (ui) => { ui.pages[0].sections[1].source.sort = 'random' } },
  { rule: 'module component props', path: '/pages/0/sections/0/props/players', change: (ui) => { ui.pages[0].sections[0].props.players = 40 } },
  { rule: 'module component takes no children', path: '/pages/0/sections/0/children', change: (ui) => { ui.pages[0].sections[0].children = [shelfOf('kid')] } },
  { rule: 'NativeView view is known', path: '/pages/0/sections/2/props/view', change: (ui) => { ui.pages[0].sections[2].props.view = 'wittyboardgames.nope' } },
  { rule: 'NativeView is last', path: '/pages/0/sections/0', change: (ui) => { ui.pages[0].sections.reverse() } },
  { rule: 'a variable the page has', path: '/pages/0/sections/1/source/players', change: (ui) => { ui.pages[0].sections[1].source.players = '$library' } },
  { rule: 'condition values', path: '/pages/0/sections/1/when/weekday/0', change: (ui) => { ui.pages[0].sections[1].when = { weekday: ['friday'] } } },
  { rule: 'condition time', path: '/pages/0/sections/1/when/timeFrom', change: (ui) => { ui.pages[0].sections[1].when = { timeFrom: '25:00' } } },
  { rule: 'a variant has a condition', path: '/pages/0/variants/0/when', change: (ui) => { ui.pages[0].variants = [{ id: 'kids', when: {}, sections: [shelfOf('kids-all')] }] } },
  { rule: 'Tab inside Tabs', path: '/pages/0/sections/1/component', change: (ui) => { ui.pages[0].sections[1] = { id: 'tab', component: 'Tab', props: { title: 'A' } } } },
  { rule: 'a notice\'s button has a label', path: '/pages/0/sections/2/children/0/props/buttonLabel', change: (ui) => { ui.pages[0].sections[2].children[0].props.to = '/page/wittyboardgames' } },
  // Inserts and presets.
  { rule: 'insert into a built-in page', path: '/inserts/0/page', change: (ui) => { ui.inserts[0].page = 'start' } },
  { rule: 'insert after a section of it', path: '/inserts/0', change: (ui) => { ui.inserts[0].after = 'jumpp' } },
  { rule: 'insert after or before', path: '/inserts/0', change: (ui) => { ui.inserts[0].before = 'hero' } },
  { rule: 'insert section ID is the module\'s', path: '/inserts/0/section/id', change: (ui) => { ui.inserts[0].section.id = 'tonight' } },
  { rule: 'insert section is valid', path: '/inserts/0/section/0/0/when/weekday/1', change: (ui) => { ui.inserts[0].section.when.weekday = ['fri', 'caturday'] } },
  { rule: 'preset ID is the module\'s', path: '/presets/0/id', change: (ui) => { ui.presets[0].id = 'recent' } },
  { rule: 'preset needs a label', path: '/presets/0/label', change: (ui) => { ui.presets[0].label = '' } },
  { rule: 'preset section is valid', path: '/presets/0/section/0/props/title', change: (ui) => { ui.presets[0].section.props.title = 7 } },
  // Navigation.
  { rule: 'entry ID is the module\'s', path: '/navigation/0/id', change: (ui) => { ui.navigation[0].id = 'boardgames' } },
  { rule: 'entry opens a page there is', path: '/navigation/0/page', change: (ui) => { ui.navigation[0].page = 'wittyboardgames-gone' } },
  { rule: 'entry is not Settings', path: '/navigation/0/screen', change: (ui) => { ui.navigation[0] = { id: 'wittyboardgames', type: 'native', screen: 'settings' } } },
  { rule: 'entry is not a library', path: '/navigation/0/type', change: (ui) => { ui.navigation[0] = { id: 'wittyboardgames', type: 'libraries' } } },
  { rule: 'entry placement', path: '/navigation/0/placement/phone', change: (ui) => { ui.navigation[0].placement.phone = 'drawer' } },
  { rule: 'entry form factor', path: '/navigation/0/placement/watch', change: (ui) => { ui.navigation[0].placement.watch = 'more' } },
  { rule: 'route entry has a label', path: '/navigation/0/label', change: (ui) => { ui.navigation[0] = { id: 'wittyboardgames', type: 'route', route: '/page/wittyboardgames' } } },
  // Settings.
  { rule: 'settings page ID is the module\'s', path: '/settings/0/id', change: (ui) => { ui.settings[0].id = 'games' } },
  { rule: 'settings page has rows', path: '/settings/0/groups', change: (ui) => { ui.settings[0].groups = [] } },
  { rule: 'setting kind', path: '/settings/0/groups/0/rows/0/kind', change: (ui) => { ui.settings[0].groups[0].rows[0].kind = 'text' } },
  { rule: 'setting key', path: '/settings/0/groups/0/rows/0/key', change: (ui) => { ui.settings[0].groups[0].rows[0].key = 'List URL' } },
  { rule: 'setting keys are unique', path: '/settings/0/groups/1/rows/0/key', change: (ui) => { ui.settings[0].groups[1].rows[0].key = 'listUrl' } },
  { rule: 'setting scope', path: '/settings/0/groups/0/rows/2/scope', change: (ui) => { ui.settings[0].groups[0].rows[2].scope = 'device' } },
  { rule: 'secret is the server\'s', path: '/settings/0/groups/0/rows/1/scope', change: (ui) => { ui.settings[0].groups[0].rows[1].scope = 'profile' } },
  { rule: 'secret has no default', path: '/settings/0/groups/0/rows/1/default', change: (ui) => { ui.settings[0].groups[0].rows[1].default = 'hunter2' } },
  { rule: 'no profile setting on an admin page', path: '/settings/0/groups/1/rows/0/scope', change: (ui) => { ui.settings[0].admin = true } },
  { rule: 'enum has options', path: '/settings/0/groups/1/rows/0/options', change: (ui) => { ui.settings[0].groups[1].rows[0].options = []; delete ui.settings[0].groups[1].rows[0].default } },
  { rule: 'only an enum has options', path: '/settings/0/groups/0/rows/2/options', change: (ui) => { ui.settings[0].groups[0].rows[2].options = [{ value: '1', label: 'One' }] } },
  { rule: 'enum default is an option', path: '/settings/0/groups/1/rows/0/default', change: (ui) => { ui.settings[0].groups[1].rows[0].default = 'adults' } },
  { rule: 'int default within min and max', path: '/settings/0/groups/0/rows/2/default', change: (ui) => { ui.settings[0].groups[0].rows[2].default = 13 } },
  { rule: 'string default within maxLength', path: '/settings/0/groups/0/rows/0/default', change: (ui) => { ui.settings[0].groups[0].rows[0].default = 'x'.repeat(301) } },
  { rule: 'info row has no value', path: '/settings/0/groups/0/rows/3', change: (ui) => { ui.settings[0].groups[0].rows.push({ key: 'about', kind: 'info', label: 'About' }) } },
]

// Renames a module name everywhere it is used.
function rename(ui, from, to) {
  const walk = (v) => {
    if (Array.isArray(v)) return v.map(walk)
    if (v && typeof v === 'object') return Object.fromEntries(Object.entries(v).map(([k, x]) => [k, walk(x)]))
    return v === from ? to : v
  }
  Object.assign(ui, walk(ui))
}
