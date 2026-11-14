// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 WittyWilla and contributors
//
// Web host API 5: what WittyMediaServer's web app gives a module's web bundle (module API 5).
// The server's web/src/moduleHost.ts is the other side, and a test there keeps the two the
// same: a name here is a promise. Adding raises the web host API; removing or changing breaks
// bundles already released.
//
// A bundle exports register(host) and returns what its integration.json claims:
//
//   import type { ModuleHost, Registration } from '@wittywilla/wms-host'
//   export function register(host: ModuleHost): Registration {
//     return { routes: { '/wittyplay/*': PlayFrame }, settings: { gaming: GamingSettings } }
//   }
import type { ComponentType, Context, ReactNode } from 'react'

export type User = { id: number; name: string; displayName?: string; isAdmin: boolean; kind: 'adult' | 'kids' | 'guest'; avatar?: string }

/** The app's own API client: the person's own sign-in, this server only. Paths start after /api. */
export interface Http {
  get<T>(path: string, signal?: AbortSignal): Promise<T>
  post<T>(path: string, body?: unknown): Promise<T>
  put<T>(path: string, body?: unknown): Promise<T>
  del<T>(path: string, body?: unknown): Promise<T>
  /** An address a <img>, <video> or <a> can use, with the sign-in a media address takes. */
  mediaUrl(path: string): string
  token(): string
  /** The server's address, or '' on the same origin. */
  base(): string
}

/** The module's own routes, /api/modules/<module><path>. */
export interface ModuleApi {
  get<T>(path: string): Promise<T>
  post<T>(path: string, body?: unknown): Promise<T>
  put<T>(path: string, body?: unknown): Promise<T>
  del<T>(path: string): Promise<T>
}

export type GameArtArgs = { itemId: string; console: string; image: string; title: string }
export type CardArgs = GameArtArgs & { kind: string }
export interface CardParts {
  art?(a: CardArgs): ReactNode | null
  detail?(a: CardArgs): ReactNode
  cover?(a: CardArgs): ReactNode
  /** Where a card's link goes instead of where the server says. */
  link?(to: string): string
}

export type RouteProps = { host: ModuleHost; params: Record<string, string | undefined> }

/** What register(host) returns. Only what integration.json claims is used. */
export interface Registration {
  routes?: Record<string, ComponentType<RouteProps>>
  cards?: Record<string, CardParts>
  players?: Record<string, ComponentType<{ host: ModuleHost; itemId: string }>>
  readers?: Record<string, ComponentType<{ host: ModuleHost }>>
  settings?: Record<string, ComponentType<{ host: ModuleHost }>>
  setup?: Record<string, ComponentType<{ host: ModuleHost; onDone: () => void }>>
}

/** The host one module is given. */
export interface ModuleHost {
  version: 5
  module: string
  /** Whether the app offers a host feature (web, frames, cards, settings, routes) at a version. */
  has(feature: string, v?: number): boolean
  http: Http
  moduleApi: ModuleApi
  /** The app's state; user is who is signed in. Other fields are the app's own and may grow. */
  useApp(): { user: User | null; version: string } & Record<string, unknown>
  useTvMode(): boolean
  toast(message: string, options?: Record<string, unknown>): void
  ask(question: Record<string, unknown>): Promise<unknown>
  /** The app's UI kit (web/src/ui): Button, IconButton, LinkButton, Field, Input, Select, Switch,
   * Textarea, Chip, Chips, Segmented, Tabs, Dialog, Sheet, Menu, Card, CardRow, EmptyState,
   * LoadError, Notice, PageSkeleton, PosterCard, Progress, ProgressRing, Shelf, Skeleton,
   * SkeletonGrid, Spinner, RowSheet, Rows, SheetRow, DataTable, Dock, KeyLegend, Tooltip,
   * useUserPref, setUserPref, getUserPref. Drawn exactly as the app draws them. */
  ui: Record<string, any> // eslint-disable-line @typescript-eslint/no-explicit-any
  /** The app's icons (web/src/icons): IconHome, IconPlay, … */
  icons: Record<string, ComponentType<{ className?: string; size?: number }>>
  ModuleMissing: ComponentType<{ ids: string[]; what?: string; onReady?: () => void }>
  LegalNote: ComponentType<{ kind: string; className?: string; newTab?: boolean }>
  sdui: {
    CardRouteContext: Context<(to: string) => string>
    GameArtContext: Context<{ art: (a: GameArtArgs) => ReactNode | null; detail?: (a: GameArtArgs) => ReactNode; cover?: (a: GameArtArgs) => ReactNode } | null>
    GlyphContext: Context<((g: 'a' | 'b' | 'x' | 'y') => ReactNode) | null>
    RowLayoutContext: Context<'shelf' | 'grid' | 'list'>
    TabsContext: Context<unknown>
    setPageRegions(r: unknown): void
    usePageRegions(): unknown
  }
  input: { focusFirst(): boolean }
}

/** globalThis.__wmsModuleHost: the shared libraries, and hostFor. A bundle never reads it
 * directly; the builder's stand-ins and register(host) do. */
export interface HostGlobal {
  api: number
  react: unknown
  reactDom: unknown
  jsxRuntime: unknown
  reactRouterDom: unknown
  wittyink?: Record<string, unknown>
  pdfjs(): Promise<unknown>
  pdfjsModule: unknown
  pdfWorkerUrl: string
  hostFor(module: string): ModuleHost
}
