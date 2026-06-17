import { HighlightStyle, StreamLanguage, syntaxHighlighting, type LanguageSupport } from '@codemirror/language'
import { json } from '@codemirror/lang-json'
import { properties } from '@codemirror/legacy-modes/mode/properties'
import { toml } from '@codemirror/legacy-modes/mode/toml'
import type { Extension } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import { tags as t } from '@lezer/highlight'
import { contentTypes } from '../../domain/types'
import type { CodeTheme } from '../../lib/settings'

type ContentTooling = {
  label: string
  language?: Extension | LanguageSupport
}

type CodeThemeDefinition = {
  dark: boolean
  background: string
  foreground: string
  gutter: string
  gutterBorder: string
  activeLine: string
  selection: string
  token: {
    keyword: string
    plain: string
    property: string
    function: string
    constant: string
    number: string
    string: string
    comment: string
    invalid: string
  }
}

const codeThemeDefinitions: Record<CodeTheme, CodeThemeDefinition> = {
  'github-light-plus': {
    dark: false,
    background: '#ffffff',
    foreground: '#24292f',
    gutter: '#57606a',
    gutterBorder: '#d8dee4',
    activeLine: '#f6f8fa',
    selection: '#0969da33',
    token: {
      keyword: '#b42318',
      plain: '#24292f',
      property: '#c2410c',
      function: '#7c3aed',
      constant: '#1d4ed8',
      number: '#4f46e5',
      string: '#0550ae',
      comment: '#57606a',
      invalid: '#82071e',
    },
  },
  'github-light': {
    dark: false,
    background: '#ffffff',
    foreground: '#24292f',
    gutter: '#57606a',
    gutterBorder: '#d8dee4',
    activeLine: '#f6f8fa',
    selection: '#0969da33',
    token: {
      keyword: '#cf222e',
      plain: '#24292f',
      property: '#953800',
      function: '#8250df',
      constant: '#0550ae',
      number: '#0550ae',
      string: '#0a3069',
      comment: '#6e7781',
      invalid: '#82071e',
    },
  },
  'one-dark': {
    dark: true,
    background: '#282c34',
    foreground: '#abb2bf',
    gutter: '#636d83',
    gutterBorder: '#3e4451',
    activeLine: '#2c313c',
    selection: '#3e4451',
    token: {
      keyword: '#c678dd',
      plain: '#abb2bf',
      property: '#e06c75',
      function: '#61afef',
      constant: '#d19a66',
      number: '#d19a66',
      string: '#98c379',
      comment: '#7f848e',
      invalid: '#f44747',
    },
  },
}

function createCodeTheme(theme: CodeTheme) {
  const definition = codeThemeDefinitions[theme]
  return [
    EditorView.theme(
      {
        '&': { backgroundColor: definition.background, color: definition.foreground },
        '.cm-content': { caretColor: definition.foreground, fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace', fontSize: '13px' },
        '.cm-gutters': { backgroundColor: definition.background, borderRight: `1px solid ${definition.gutterBorder}`, color: definition.gutter },
        '.cm-activeLine': { backgroundColor: definition.activeLine },
        '.cm-activeLineGutter': { backgroundColor: definition.activeLine },
        '.cm-selectionBackground': { backgroundColor: `${definition.selection} !important` },
        '&.cm-focused': { outline: 'none' },
      },
      { dark: definition.dark },
    ),
    syntaxHighlighting(HighlightStyle.define([
      { tag: [t.keyword, t.operatorKeyword], color: definition.token.keyword },
      { tag: [t.name, t.deleted, t.character, t.macroName], color: definition.token.plain },
      { tag: [t.propertyName, t.variableName, t.labelName], color: definition.token.property },
      { tag: [t.function(t.variableName), t.function(t.propertyName)], color: definition.token.function },
      { tag: [t.color, t.constant(t.name), t.standard(t.name)], color: definition.token.constant },
      { tag: [t.definition(t.name), t.separator], color: definition.token.plain },
      { tag: [t.brace, t.bracket, t.paren], color: definition.token.plain },
      { tag: [t.annotation], color: definition.token.function },
      { tag: [t.number, t.bool, t.null, t.atom, t.special(t.variableName)], color: definition.token.number },
      { tag: [t.string, t.special(t.string), t.regexp, t.inserted], color: definition.token.string },
      { tag: [t.comment], color: definition.token.comment },
      { tag: [t.invalid], color: definition.token.invalid },
    ])),
  ]
}

const contentTypeNameKeys = new Map<string, number>([
  ['JSON', 1],
  ['TOML', 2],
  ['INI', 3],
  ['PLAINTEXT', 4],
])

function contentTypeKey(contentType?: number | string) {
  const numeric = Number(contentType)
  if (Number.isFinite(numeric)) return numeric
  if (typeof contentType === 'string') return contentTypeNameKeys.get(contentType.toUpperCase())
  return undefined
}

const registry = new Map<number, ContentTooling>([
  [1, { label: 'JSON', language: json() }],
  [2, { label: 'TOML', language: StreamLanguage.define(toml) }],
  [3, { label: 'INI', language: StreamLanguage.define(properties) }],
  [4, { label: 'PLAINTEXT' }],
])

export function getContentTooling(contentType?: number | string) {
  const key = contentTypeKey(contentType)
  if (key !== undefined) return registry.get(key) || { label: String(contentType || '-') }

  return { label: String(contentType || '-') }
}

export function getContentTypeLabel(contentType?: number | string) {
  const key = contentTypeKey(contentType)
  return getContentTooling(contentType).label || contentTypes.find((item) => item.value === key)?.label || String(contentType || '-')
}

export function getContentLanguage(contentType?: number | string) {
  const language = getContentTooling(contentType).language
  return language ? [language] : []
}

export function getContentExtensions(contentType?: number | string, codeTheme: CodeTheme = 'github-light-plus', lineWrapping = true) {
  return [...(lineWrapping ? [EditorView.lineWrapping] : []), ...createCodeTheme(codeTheme), ...getContentLanguage(contentType)]
}
