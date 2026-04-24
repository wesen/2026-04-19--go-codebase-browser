import { BaseQueryFn } from '@reduxjs/toolkit/query';

declare global {
  interface Window {
    Go: new () => {
      importObject: WebAssembly.Imports;
      run(instance: WebAssembly.Instance): void;
    };
    codebaseBrowser?: {
      initWasm: (...args: string[]) => string;
      findSymbols: (query: string, kind: string) => string;
      getSymbol: (id: string) => string;
      getXref: (id: string) => string;
      getSnippet: (id: string, kind: string) => string;
      getPackages: () => string;
      getIndexSummary: () => string;
      getDocPages: () => string;
      getDocPage: (slug: string) => string;
    };
  }
}

let wasmReady = false;
let initError: string | null = null;

// Cached precomputed data for direct lookups (snippetRefs, sourceRefs, fileXref)
let precomputedCache: Record<string, unknown> | null = null;

export function isWasmReady(): boolean {
  return wasmReady;
}

export function getInitError(): string | null {
  return initError;
}

export async function getPrecomputed(): Promise<Record<string, unknown>> {
  if (precomputedCache) return precomputedCache;
  const resp = await fetch('/precomputed.json');
  precomputedCache = await resp.json();
  return precomputedCache!;
}

export async function initWasm(wasmPath = '/search.wasm'): Promise<void> {
  if (wasmReady) return;
  if (!window.Go) {
    throw new Error('Go WASM runtime not loaded. Include wasm_exec.js before loading this module.');
  }

  const go = new window.Go();

  let response: Response;
  if (typeof WebAssembly.instantiateStreaming === 'function') {
    response = await fetch(wasmPath);
    const result = await WebAssembly.instantiateStreaming(Promise.resolve(response), go.importObject);
    go.run(result.instance);
  } else {
    const resp = await fetch(wasmPath);
    const bytes = await resp.arrayBuffer();
    const result = await WebAssembly.instantiate(bytes, go.importObject);
    go.run(result.instance);
  }

  // Wait for the global to appear (Go WASM sets it asynchronously)
  let attempts = 0;
  while (!window.codebaseBrowser && attempts < 50) {
    await new Promise((r) => setTimeout(r, 50));
    attempts++;
  }
  if (!window.codebaseBrowser) {
    throw new Error('WASM exports not available after loading');
  }

  // Load precomputed data
  const precomputed = await fetch('/precomputed.json').then((r) => r.json());

  const result = window.codebaseBrowser.initWasm(
    JSON.stringify(precomputed),
    JSON.stringify(precomputed.searchIndex || {}),
    JSON.stringify(precomputed.xrefIndex || {}),
    JSON.stringify(precomputed.snippets || {}),
    JSON.stringify(precomputed.docManifest || []),
    JSON.stringify(precomputed.docHTML || {})
  );

  if (result !== 'ok') {
    throw new Error('WASM init failed: ' + result);
  }

  wasmReady = true;
}

// RTK-Query baseQuery that routes to WASM instead of HTTP
export const wasmBaseQuery: BaseQueryFn<string, unknown, { status: string; data?: string }> = async (
  arg
) => {
  if (!wasmReady) {
    await initWasm();
  }
  if (!window.codebaseBrowser) {
    return { error: { status: 'WASM_ERROR', data: 'WASM not initialized' } };
  }

  try {
    const endpoint = arg as string;
    let result: string;

    switch (endpoint) {
      case 'index':
        result = window.codebaseBrowser.getIndexSummary();
        break;
      case 'packages':
        result = window.codebaseBrowser.getPackages();
        break;
      default: {
        if (endpoint.startsWith('symbol:')) {
          result = window.codebaseBrowser.getSymbol(endpoint.slice(7));
        } else if (endpoint.startsWith('search:')) {
          const [q, kind] = endpoint.slice(7).split('|');
          result = window.codebaseBrowser.findSymbols(q || '', kind || '');
        } else if (endpoint.startsWith('xref:')) {
          result = window.codebaseBrowser.getXref(endpoint.slice(5));
        } else if (endpoint.startsWith('snippet:')) {
          const [sym, kind] = endpoint.slice(8).split('|');
          result = window.codebaseBrowser.getSnippet(sym, kind || 'declaration');
        } else if (endpoint.startsWith('docPages')) {
          result = window.codebaseBrowser.getDocPages();
        } else if (endpoint.startsWith('docPage:')) {
          result = window.codebaseBrowser.getDocPage(endpoint.slice(8));
        } else {
          return { error: { status: 'UNKNOWN_ENDPOINT', data: endpoint } };
        }
        break;
      }
    }

    return { data: JSON.parse(result) };
  } catch (err) {
    return { error: { status: 'WASM_ERROR', data: String(err) } };
  }
};
