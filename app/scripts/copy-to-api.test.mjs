import assert from 'node:assert/strict';
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';

import { copyBundle } from './copy-to-api.mjs';

async function sandbox() {
  const root = await mkdtemp(join(tmpdir(), 'screen-share-copy-'));
  return {
    root,
    source: join(root, 'source'),
    destination: join(root, 'destination'),
    cleanup: () => rm(root, { recursive: true, force: true }),
  };
}

test('rejects a source without index.html', async (t) => {
  const box = await sandbox();
  t.after(box.cleanup);
  await mkdir(box.source, { recursive: true });
  await assert.rejects(copyBundle(box.source, box.destination), /index\.html/);
});

test('cleans stale files and copies the bundle recursively', async (t) => {
  const box = await sandbox();
  t.after(box.cleanup);
  await mkdir(join(box.source, 'assets'), { recursive: true });
  await mkdir(box.destination, { recursive: true });
  await writeFile(join(box.source, 'index.html'), '<html>ok</html>');
  await writeFile(join(box.source, 'assets', 'icon.svg'), '<svg/>');
  await writeFile(join(box.destination, 'stale.js'), 'stale');

  await copyBundle(box.source, box.destination);

  assert.equal(await readFile(join(box.destination, 'index.html'), 'utf8'), '<html>ok</html>');
  assert.equal(await readFile(join(box.destination, 'assets', 'icon.svg'), 'utf8'), '<svg/>');
  await assert.rejects(readFile(join(box.destination, 'stale.js')), /ENOENT/);
});

test('rejects secret and database files', async (t) => {
  const box = await sandbox();
  t.after(box.cleanup);
  await mkdir(box.source, { recursive: true });
  await writeFile(join(box.source, 'index.html'), '<html>ok</html>');
  await writeFile(join(box.source, '.env'), 'LINK_ID_SALT=secret');
  await assert.rejects(copyBundle(box.source, box.destination), /forbidden/i);
});
