// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors

// The server's example module (module-boardgames.json beside this file), written with
// the kit's helpers: write-sdui.mjs must make exactly that file from it.
import { defineModuleUI, moduleUI, nativeView, notice, section, shelf } from '../../wittymodule-sdui.ts'

export default defineModuleUI({
  ...moduleUI('wittyboardgames', { theme: { label: 'Board games', tokens: { accent: '#2E7D32' } } }),
  sources: [
    {
      type: 'wittyboardgames.shelf',
      label: 'Board games on the shelf',
      description: "The household's board games, from the module's own list.",
      icon: 'game',
      params: [
        { name: 'players', type: 'int', label: 'Players', help: 'Only games for this many players; 0 for any.', min: 0, max: 12, since: 2 },
        { name: 'sort', type: 'enum', label: 'Order', default: 'name', options: [{ value: 'name', label: 'By name' }, { value: 'played', label: 'Last played first' }, { value: 'shuffle', label: 'Shuffled' }], since: 2 },
        { name: 'limit', type: 'int', label: 'At most', min: 1, max: 100, since: 2 },
      ],
      yields: ['game'],
      cardStyle: 'square',
      paged: true,
      seeAll: '/page/wittyboardgames',
      title: 'Board games',
      since: 2,
    },
  ],
  components: [
    {
      type: 'wittyboardgames.GameNight',
      label: 'Game night',
      description: 'A row of games picked for tonight\'s players, shuffled every time.',
      icon: 'game',
      props: [
        { name: 'title', type: 'text', label: 'Title', default: 'Game night', since: 2 },
        { name: 'players', type: 'int', label: 'Players', default: 4, min: 1, max: 12, since: 2 },
      ],
      section: shelf('night', { type: 'wittyboardgames.shelf', players: '{{players}}', sort: 'shuffle', limit: 12 }, { title: '{{title}} for {{players}}', cardSize: 'l', seeAll: 'always' }),
    },
  ],
  views: [{ id: 'wittyboardgames.scoreboard', label: 'Scoreboard', description: "The module's live scoreboard, drawn by its web bundle.", platforms: ['web', 'desktop'] }],
  pages: [
    {
      id: 'wittyboardgames',
      kind: 'custom',
      title: 'Board games',
      icon: 'game',
      sections: [
        section('night', 'wittyboardgames.GameNight', { props: { players: 4 } }),
        shelf('all', { type: 'wittyboardgames.shelf', sort: 'name' }, { title: 'Every game', cardStyle: 'square', seeAll: 'never' }),
        nativeView('scores', 'wittyboardgames.scoreboard', [notice('scores-note', 'The scoreboard is on the web app.', { tone: 'info' })]),
      ],
    },
  ],
  inserts: [{ page: 'home', after: 'jump', section: section('wittyboardgames-tonight', 'wittyboardgames.GameNight', { when: { weekday: ['fri', 'sat'] } }) }],
  presets: [{ id: 'wittyboardgames-recent', label: 'Board games played lately', section: shelf('boardgames', { type: 'wittyboardgames.shelf', sort: 'played', limit: 20 }, { title: 'Played lately' }) }],
  navigation: [{ id: 'wittyboardgames', type: 'page', page: 'wittyboardgames', placement: { phone: 'more', tv: 'more' } }],
  settings: [
    {
      id: 'wittyboardgames',
      title: 'Board games',
      icon: 'game',
      groups: [
        {
          title: 'The shelf',
          rows: [
            { key: 'listUrl', kind: 'string', label: 'Where the list is', help: 'A web address of a BoardGameGeek collection, or empty for the list kept here.', maxLength: 300 },
            { key: 'apiKey', kind: 'secret', label: 'API key' },
            { key: 'household', kind: 'int', label: 'Players in the house', default: 4, min: 1, max: 12 },
          ],
        },
        {
          title: 'For you',
          footer: 'Each profile chooses its own.',
          rows: [{ key: 'ages', kind: 'enum', scope: 'profile', label: 'Games for', default: 'all', options: [{ value: 'all', label: 'Everyone' }, { value: 'kids', label: 'Children' }] }],
        },
      ],
    },
  ],
})
