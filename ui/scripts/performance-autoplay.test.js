import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import vm from 'node:vm';
import ts from 'typescript';
import { fileURLToPath } from 'node:url';
import dayjs from 'dayjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const uiRoot = path.resolve(__dirname, '..');

const loadTranspiledCommonJsModule = (relativeEntry) => {
  const entry = path.resolve(uiRoot, relativeEntry);
  const source = fs.readFileSync(entry, 'utf8');
  const transpiled = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2020,
    },
    fileName: entry,
  });

  const module = { exports: {} };
  const sandbox = {
    module,
    exports: module.exports,
    require: (specifier) => {
      if (specifier === 'dayjs') {
        return { default: dayjs };
      }
      throw new Error(`Unexpected require in test transpile sandbox: ${specifier}`);
    },
    console,
  };
  vm.runInNewContext(transpiled.outputText, sandbox, { filename: entry });
  return module.exports;
};

const { shouldAutoplayPerformanceMessage } = loadTranspiledCommonJsModule('src/components/chat/performanceAutoplay.ts');

const now = dayjs('2026-06-12T22:40:00+08:00');
const freshTs = now.subtract(3, 'second').valueOf();
const staleTs = now.subtract(20, 'second').valueOf();

assert.equal(shouldAutoplayPerformanceMessage(freshTs, true, 'sent', now), true);
assert.equal(shouldAutoplayPerformanceMessage(freshTs, false, 'sent', now), true);
assert.equal(shouldAutoplayPerformanceMessage(String(freshTs), true, 'sent', now), true);
assert.equal(shouldAutoplayPerformanceMessage(now.subtract(3, 'second').toISOString(), true, 'sent', now), true);
assert.equal(shouldAutoplayPerformanceMessage(staleTs, true, 'sent', now), false);
assert.equal(shouldAutoplayPerformanceMessage(staleTs, false, 'sent', now), false);
assert.equal(shouldAutoplayPerformanceMessage('', true, 'sent', now), false);
assert.equal(shouldAutoplayPerformanceMessage(freshTs, true, 'sending', now), true);

console.log('performance autoplay regressions passed');
