import * as fs from 'fs';
import { extract } from './extract.js';

interface Args {
  moduleRoot: string;
  tsconfig?: string;
  moduleName?: string;
  pathPrefix?: string;
  out: string;
  pretty: boolean;
}

function parseArgs(argv: string[]): Args {
  const args: Partial<Args> = { out: '-', pretty: true };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--module-root') args.moduleRoot = argv[++i];
    else if (a === '--tsconfig') args.tsconfig = argv[++i];
    else if (a === '--module-name') args.moduleName = argv[++i];
    else if (a === '--path-prefix') args.pathPrefix = argv[++i];
    else if (a === '--out' || a === '-o') args.out = argv[++i];
    else if (a === '--compact') args.pretty = false;
    else if (a === '--help' || a === '-h') {
      printUsage();
      process.exit(0);
    } else if (!args.moduleRoot) args.moduleRoot = a;
    else throw new Error(`unexpected argument: ${a}`);
  }
  if (!args.moduleRoot) {
    throw new Error('missing --module-root (or positional arg)');
  }
  return args as Args;
}

function printUsage() {
  process.stderr.write(
    `Usage: ts-indexer --module-root <path> [--tsconfig <file>] [--module-name <name>]\n` +
      `                  [--path-prefix <p>] [--out <file|-> ] [--compact]\n`,
  );
}

function main() {
  let args: Args;
  try {
    args = parseArgs(process.argv.slice(2));
  } catch (err) {
    process.stderr.write(`ts-indexer: ${(err as Error).message}\n`);
    printUsage();
    process.exit(2);
  }
  const idx = extract({
    moduleRoot: args.moduleRoot,
    tsconfig: args.tsconfig,
    moduleName: args.moduleName,
    pathPrefix: args.pathPrefix,
  });
  const text = args.pretty ? JSON.stringify(idx, null, 2) : JSON.stringify(idx);
  if (args.out === '-' || !args.out) {
    process.stdout.write(text + '\n');
  } else {
    fs.writeFileSync(args.out, text + '\n');
  }
}

main();
