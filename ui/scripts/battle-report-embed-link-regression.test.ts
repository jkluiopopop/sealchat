import assert from 'node:assert/strict'
import {
  parseSingleBattleReportEmbedLinkText,
} from '../src/utils/battleReportEmbedLink'

assert.deepEqual(
  parseSingleBattleReportEmbedLinkText('/#/EWIJ5Fa1y4NG4tui/K6IWS5kmOP64H1IT?battleReport=FtwAvDVBYQmIjQuM1'),
  {
    worldId: 'EWIJ5Fa1y4NG4tui',
    channelId: 'K6IWS5kmOP64H1IT',
    reportId: 'FtwAvDVBYQmIjQuM1',
    rawLink: '/#/EWIJ5Fa1y4NG4tui/K6IWS5kmOP64H1IT?battleReport=FtwAvDVBYQmIjQuM1',
  },
  '战报嵌入链接应支持相对 hash 路由',
)

assert.deepEqual(
  parseSingleBattleReportEmbedLinkText('https://example.test/#/world_1/channel-2?battleReport=report_3'),
  {
    worldId: 'world_1',
    channelId: 'channel-2',
    reportId: 'report_3',
    rawLink: 'https://example.test/#/world_1/channel-2?battleReport=report_3',
  },
  '战报嵌入链接应继续支持完整 URL',
)

console.log('battle-report embed link regressions passed')
