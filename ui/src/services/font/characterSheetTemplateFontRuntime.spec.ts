import {
  buildCharacterSheetFontCssMessage,
  parseCharacterSheetFontManifest,
  selectPlatformFontChunksForText,
} from './characterSheetTemplateFontRuntime'

const html = `
<script type="application/sealchat-fonts+json">
{
  "version": 1,
  "global": "body",
  "fonts": [
    { "key": "body", "platformFontId": "font_body", "cssVar": "--font-body" },
    { "key": "title", "platformFontId": "font_title", "cssVar": "--font-title" }
  ]
}
</script>`

const manifest = parseCharacterSheetFontManifest(html)
const selected = selectPlatformFontChunksForText([
  { name: 'latin.woff2', url: '/latin.woff2', unicodeRange: 'U+0000-00FF' },
  { name: 'cjk.woff2', url: '/cjk.woff2', unicodeRange: 'U+4E00-9FFF' },
], '调查员A')
const message = buildCharacterSheetFontCssMessage('body{font-family:var(--font-body)}')

if (manifest.fonts.length !== 2 || selected.length !== 2 || message.type !== 'SEALCHAT_FONT_CSS') {
  throw new Error('character sheet font runtime smoke check failed')
}
