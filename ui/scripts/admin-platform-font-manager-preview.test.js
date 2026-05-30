const fs = require('node:fs')
const path = require('node:path')

const file = path.resolve(__dirname, '../src/views/admin/components/AdminPlatformFontManager.vue')
const source = fs.readFileSync(file, 'utf8')

const required = [
  'ensurePlatformFontAssetLoaded',
  'copyTextWithResult',
  'platform-font-manager__font-id',
  'platform-font-manager__preview-block',
  '复制 ID',
  '字体 ID',
]

const missing = required.filter((token) => !source.includes(token))

if (missing.length > 0) {
  console.error(`AdminPlatformFontManager preview/id UI missing: ${missing.join(', ')}`)
  process.exit(1)
}
