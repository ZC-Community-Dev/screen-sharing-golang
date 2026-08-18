import { cp, readdir, rm, stat } from 'node:fs/promises';
import { dirname, extname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const forbidden = (name) =>
  name === '.env' ||
  name.startsWith('.env.') ||
  extname(name).toLowerCase() === '.db' ||
  name.toLowerCase().includes('link_id_salt');

async function validateTree(root, current = root) {
  for (const entry of await readdir(current, { withFileTypes: true })) {
    if (forbidden(entry.name)) {
      throw new Error(`forbidden file in frontend bundle: ${entry.name}`);
    }
    if (entry.isDirectory()) {
      await validateTree(root, join(current, entry.name));
    }
  }
}

export async function copyBundle(source, destination) {
  const indexPath = join(source, 'index.html');
  let index;
  try {
    index = await stat(indexPath);
  } catch {
    throw new Error('frontend build is missing index.html');
  }
  if (!index.isFile()) {
    throw new Error('frontend build index.html is not a file');
  }

  await validateTree(source);
  await rm(destination, { recursive: true, force: true });
  await cp(source, destination, { recursive: true, force: true });
}

async function main() {
  const scriptDir = dirname(fileURLToPath(import.meta.url));
  const source = resolve(scriptDir, '..', 'dist', 'app', 'browser');
  const destination = resolve(scriptDir, '..', '..', 'api', 'internal', 'web', 'dist', 'browser');
  await copyBundle(source, destination);
  console.log(`Copied Angular bundle to ${destination}`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  main().catch((error) => {
    console.error(error.message);
    process.exitCode = 1;
  });
}
