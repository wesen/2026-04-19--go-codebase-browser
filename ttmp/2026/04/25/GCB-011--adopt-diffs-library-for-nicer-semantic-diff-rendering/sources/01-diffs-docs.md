---
Title: Diffs docs source snapshot
Ticket: GCB-011
Status: active
Topics:
  - codebase-browser
  - react-frontend
  - semantic-diff
  - ui-design
  - documentation-tooling
DocType: reference
Intent: reference
Owners: []
RelatedFiles: []
ExternalSources:
  - https://diffs.com/docs
Summary: Defuddle-extracted Markdown snapshot of the Diffs documentation.
LastUpdated: 2026-04-25T12:35:00-04:00
WhatFor: Offline/reference copy of upstream @pierre/diffs documentation for the GCB-011 integration.
WhenToUse: Use when implementing Diffs support without repeatedly fetching online docs.
---

## Overview

Diffs is in early active development—APIs are subject to change.

**Diffs** is a library for rendering code and diffs on the web. This includes both high-level, easy-to-use components, as well as exposing many of the internals if you want to selectively use specific pieces. We've built syntax highlighting on top of [Shiki](https://shiki.style/) which provides a lot of great theme and language support.

We have an opinionated stance in our architecture: **browsers are rather efficient at rendering raw HTML**. We lean into this by having all the lower level APIs purely rendering strings (the raw HTML) that are then consumed by higher-order components and utilities. This gives us great performance and flexibility to support popular libraries like React as well as provide great tools if you want to stick to vanilla JavaScript and HTML. The higher-order components render all this out into Shadow DOM and CSS grid layout.

Generally speaking, you're probably going to want to use the higher level components since they provide an easy-to-use API that you can get started with rather quickly. We currently only have components for vanilla JavaScript and React, but will add more if there's demand.

For this overview, we'll talk about the vanilla JavaScript components for now but there are React equivalents for all of these.

## Rendering Diffs

Our goal with visualizing diffs was to provide some flexible and approachable APIs for *how* you may want to render diffs. For this, we provide a component called `FileDiff`.

There are two ways to render diffs with `FileDiff`:

1. Provide two versions of a file or code snippet to compare
2. Consume a patch file

You can see examples of these approaches below, in both JavaScript and React.

## Merge conflict resolution UI

Render conflicts through a dedicated diff primitive that treats current and incoming sections as structured additions/deletions without running text diffing. Resolve by choosing current, incoming, or both changes and preview the updated file instantly.

## Installation

Diffs is [published as an npm package](https://www.npmjs.com/package/@pierre/diffs). Install Diffs with the package manager of your choice:

### Package Exports

The package provides several entry points for different use cases:

| Package | Description |
| --- | --- |
| `@pierre/diffs` | [Vanilla JS components](https://diffs.com/docs#vanilla-js-api) and [utility functions](https://diffs.com/docs#utilities) for parsing and rendering diffs |
| `@pierre/diffs/react` | [React components](https://diffs.com/docs#react-api) for rendering diffs with full interactivity |
| `@pierre/diffs/ssr` | [Server-side rendering utilities](https://diffs.com/docs#ssr) for pre-rendering diffs with syntax highlighting |
| `@pierre/diffs/worker` | [Worker pool utilities](https://diffs.com/docs#worker-pool) for offloading syntax highlighting to background threads |

## Core Types

Before diving into the components, it's helpful to understand the two core data structures used throughout the library.

### FileContents

`FileContents` represents a single file. Use it when rendering a file with the `<File>` component, or pass two of them as `oldFile` and `newFile` to diff components.

`FileDiffMetadata` represents the differences between two files. It contains the hunks (changed regions), line counts, and optionally the full file contents for expansion.

**Tip:** You can generate `FileDiffMetadata` using [`parseDiffFromFile`](https://diffs.com/docs#utilities-parsedifffromfile) (from two file versions) or [`parsePatchFiles`](https://diffs.com/docs#utilities-parsepatchfiles) (from a patch string).

### Creating Diffs

There are two ways to create a `FileDiffMetadata`.

#### From Two Files

Use `parseDiffFromFile` when you have both file versions. This approach includes the full file contents, enabling the "expand unchanged" feature.

#### From a Patch String

Use `parsePatchFiles` when you have a unified diff or patch file. This is useful when working with git output or patch files from APIs.

**Tip:** If you need to change the language after creating a `FileContents` or `FileDiffMetadata`, use the [`setLanguageOverride`](https://diffs.com/docs#utilities-setlanguageoverride) utility function.

## React API

Import React components from `@pierre/diffs/react`.

We offer a variety of components to render diffs and files. Many of them share similar types of props, which you can find documented in [Shared Props](https://diffs.com/docs#react-api-shared-props).

### Components

The React API exposes five main components:

- `MultiFileDiff` compares two file versions
- `PatchDiff` renders from a patch string
- `FileDiff` renders a pre-parsed `FileDiffMetadata`
- `File` renders a single code file without a diff
- `UnresolvedFile` renders merge conflict markers with built-in resolution UI
	- *Currently in beta/experimental and may change in future releases.*

`UnresolvedFile` is intentionally uncontrolled in React. Treat `file` as initial input and remount (for example, with a changing `key`) when you want to reset.

### Shared Props

The three diff components (`MultiFileDiff`, `PatchDiff`, and `FileDiff`) share a common set of props for configuration, annotations, and styling. The `File` component has similar props, but uses `LineAnnotation` instead of `DiffLineAnnotation` (no `side` property).

Header customization and collapsing behavior:

- Use `renderHeaderPrefix` to render custom UI at the beginning of the built-in header, before the filename and icons, while keeping the default header layout.
- Use `renderHeaderMetadata` to render custom UI at the end of the built-in header, after the diff stats, while keeping the default header layout.
- Use `renderCustomHeader` when you want to replace the built-in header content with your own custom designed one.
- For diff components, these header callbacks receive `fileDiff: FileDiffMetadata`.
- For `File`, the corresponding header callbacks receive `file: FileContents`.
- Use `options.collapsed` to hide file body content while keeping the file header visible.

Token callbacks (`onTokenClick`, `onTokenEnter`, `onTokenLeave`) and `useTokenTransformer` are documented in [Token Hooks](https://diffs.com/docs#token-hooks), including examples, payload details, performance notes, and Worker Pool caveats.

## Vanilla JS API

Import vanilla JavaScript classes, components, and methods from `@pierre/diffs`.

### Components

The Vanilla JS API exposes three core components: `FileDiff` (compare two file versions or render a pre-parsed `FileDiffMetadata`), `File` (render a single code file without diff), and `UnresolvedFile` (render merge conflicts with built-in resolution controls). Typically you'll want to interface with these as they'll handle all the complicated aspects of syntax highlighting, theming, and full interactivity for you.

> `UnresolvedFile` is currently beta/experimental and may change in future releases.

`UnresolvedFile` in vanilla supports both uncontrolled and controlled callbacks (`onMergeConflictResolve` / `onMergeConflictAction`).

### Props

Both `FileDiff` and `File` accept an options object in their constructor. The `File` component has similar options, but excludes diff-specific settings and uses `LineAnnotation` instead of `DiffLineAnnotation` (no `side` property).

Header customization and collapsing behavior:

- Use `renderHeaderPrefix` to render custom UI at the beginning of the built-in `FileDiff` header, before the filename and icon, while keeping the default header layout.
- Use `renderHeaderMetadata` to render custom UI at the end of the built-in `FileDiff` header, after the diff stats, while keeping the default header layout.
- Use `renderCustomHeader` when you want to replace the built-in header content entirely.
- In `File`, header callbacks receive `file: FileContents`.
- Use `collapsed` in constructor options to hide file body content while keeping the file header visible.

Token callbacks (`onTokenClick`, `onTokenEnter`, `onTokenLeave`) and `useTokenTransformer` are documented in [Token Hooks](https://diffs.com/docs#token-hooks), including examples, payload details, performance notes, and Worker Pool caveats.

#### Custom Hunk Separators

Start with the [Hunk Separators](https://diffs.com/docs#hunk-separators) section first. In most cases, styling the built-in separator markup with `unsafeCSS` is the better approach.

If that is still not enough, the low-level `hunkSeparators(hunkData, instance)` function remains available in Vanilla JS as a last-resort escape hatch. It is being phased out and is not the recommended path for new integrations, but the example below shows how it works when you truly need to render your own elements:

### Renderers

For most use cases, you should use the higher-level components like `FileDiff` and `File` (vanilla JS) or the React components (`MultiFileDiff`, `FileDiff`, `PatchDiff`, `File`). These renderers are low-level building blocks intended for advanced use cases.

These renderer classes handle the low-level work of parsing and rendering code with syntax highlighting. Useful when you need direct access to the rendered output as [HAST](https://github.com/syntax-tree/hast) nodes or HTML strings for custom rendering pipelines.

#### DiffHunksRenderer

Takes a `FileDiffMetadata` data structure and renders out the raw HAST (Hypertext Abstract Syntax Tree) elements for diff hunks. You can generate `FileDiffMetadata` via `parseDiffFromFile` or `parsePatchFiles` utility functions.

#### FileRenderer

Takes a `FileContents` object (just a filename and contents string) and renders syntax-highlighted code as HAST elements. Useful for rendering single files without any diff context.

## Virtualization

Virtualization is in beta and subject to change. It is opt-in and best used when rendering very large files or many diffs in one scroll view.

Virtualization in Diffs uses estimated line and file heights to keep large renders fast. It renders placeholders and spacer buffers for off-screen content, then renders visible lines in hunk-sized batches as you scroll.

Internally, the virtualizer listens to scroll and resize updates, computes a window with overscan, and reconciles measured DOM heights after render. This keeps scroll position stable even when line heights change because of wrapped lines or annotations.

For best results, you'll need to pass a `metrics` config object to your files or diffs when your layout differs from the defaults. These metrics help the Virtualizer estimate file and diff sizes more accurately before content is measured. For large diffs, using virtualization with a [Worker Pool](https://diffs.com/docs#worker-pool) is strongly recommended.

### Getting Started

To use virtualization, start with a scrollable container (an HTML element or the window). Directly inside that container, add a content wrapper that holds all diff/file instances and any other content you render. The virtualizer uses this wrapper to track content size changes.

Inside that scroll container, render the `VirtualizedFile` and `VirtualizedFileDiff` components. In React, this is handled automatically by the built-in `Virtualizer` context. In vanilla JS, you manage this explicitly by creating a `Virtualizer` instance and wiring it to `VirtualizedFile` / `VirtualizedFileDiff` instead of the traditional APIs.

These APIs are still early and are subject to change. We may merge related virtualization functionality into the top-level `File` and `FileDiff` components before shipping, so please share feedback as you test these APIs.

### React

In React, wrap your diff/file components in `Virtualizer` from `@pierre/diffs/react`. The `Virtualizer` component is your scroll container. Currently, the React wrapper does not support window scrolling unless you orchestrate your own provider via `VirtualizerContext.Provider` (from `@pierre/diffs/react`) and pass a manually created `Virtualizer` instance (from `@pierre/diffs`).

You can tune virtualization behavior with the `config` prop.

`Virtualizer` props:

- `config`: partial virtualizer config (`overscrollSize`, `intersectionObserverMargin`, `resizeDebugging`)
- `className` / `style`: applied to the outer scroll root
- `contentClassName` / `contentStyle`: applied to the inner content wrapper

### Vanilla JavaScript

In vanilla JS, create a `Virtualizer` instance and pass it into `VirtualizedFileDiff` or `VirtualizedFile`.

### Notes

- Prefer virtualization for very large files or long diff lists any sort of scenario where it's hard to anticipate the constraints of the of the scroll view
- Keep `metrics` aligned with your layout if you customize heights.
- Use `resizeDebugging` with the `Virtualizer` temporarily when tuning metrics, and to confirm everything is working properly. Don't forget to disable it in production.

## Hunk Separators

The `hunkSeparators` option controls how collapsed unchanged regions are displayed. For customization, we recommend starting with a built-in preset and layering `unsafeCSS` on top.

Passing a render function is only documented for the Vanilla JS APIs. It is being phased out, does not work well with the container-managed and virtualization-oriented React APIs, and is not compatible with SSR. We strongly recommend avoiding that path and customizing built-in separators with `unsafeCSS` instead.

The Custom CSS example below keeps the built-in `line-info-basic` markup and tweaks it with CSS.

- blends the separator row with the diff background
- hunk content and controls are rendered in every gutter and content region, but the custom CSS targets only the left-most gutter elements
- aligns the arrow glyphs with the number column
- replaces the built-in SVG icon with CSS-generated arrows
- renders the `Expand All` button, which is normally hidden by default

### Built-in Types

- `line-info`: Rounded corner separator with collapsed line count and expansion controls.
- `line-info-basic`: Compact, full-width variant of `line-info` with expansion controls.
- `metadata`: Patch-style separator (`@@ -x,y +a,b @@`) with no expansion controls.
- `simple`: Minimal separator bar.

### Custom CSS Example

If CSS hooks are not enough, the low-level `hunkSeparators(hunkData,   instance)` function still exists on the Vanilla JS `FileDiff` API. We only recommend that escape hatch as a last resort. It is being phased out, and it does not fit the container-managed and virtualization-oriented APIs that the React components rely on.

## Utilities

Import utility functions from `@pierre/diffs`. These can be used with any framework or rendering approach.

### diffAcceptRejectHunk

Programmatically accept or reject individual hunks (or specific change blocks inside a hunk) in a diff. This is useful for building interactive code review interfaces, AI-assisted coding tools, or any workflow where users need to selectively apply changes.

To resolve an entire hunk, pass `'accept'`, `'reject'`, or `'both'`. To resolve only one change block in a hunk, pass an object with `type` and `changeIndex` (for example: `diffAcceptRejectHunk(diff, hunkIndex, { type: 'accept', changeIndex: 0 })`). `changeIndex` maps to the target entry in that hunk's `hunkContent` array.

When you **accept** a hunk, the new (additions) version is kept and the hunk is converted to context lines. When you **reject** a hunk, the old (deletions) version is restored. You can also use **both** as a lower-level way to mux the two sides together, which keeps the old lines first and then appends the new lines before collapsing the result back to context. The function returns a new `FileDiffMetadata` object with all line numbers properly adjusted for subsequent hunks.

### resolveMergeConflict

Apply a merge conflict action payload to a file string and return the next contents.

> **Experimental:** `UnresolvedFile` -related merge conflict APIs are currently beta/experimental and may change in future releases.

Default merge-conflict buttons work even without callbacks: `UnresolvedFile` applies the resolution internally.

In vanilla, provide `onMergeConflictAction` for controlled state (for example, to persist resolved contents, sync external stores, or trigger side effects). Use `onMergeConflictResolve` when you want uncontrolled resolution plus a notification with the resolved file. React `UnresolvedFile` is intentionally uncontrolled.

### disposeHighlighter

Dispose the shared Shiki highlighter instance to free memory. Useful when cleaning up resources in single-page applications.

### getSharedHighlighter

Get direct access to the shared Shiki highlighter instance used internally by all components. Useful for custom highlighting operations.

### parseDiffFromFile

Compare two versions of a file and generate a `FileDiffMetadata` structure. Use this when you have the full contents of both file versions rather than a patch string.

If both `oldFile` and `newFile` have a `cacheKey`, the resulting `FileDiffMetadata` will automatically receive a combined cache key (format: `oldKey:newKey`). See [Render Cache](https://diffs.com/docs#worker-pool-render-cache) for more information.

An optional `throwOnError` parameter (default: `false`) controls error handling. When `true`, parsing errors throw exceptions; when `false`, errors are logged to the console and parsing continues on a best-effort basis.

### parsePatchFiles

Parse unified diff / patch file content into structured data. Handles both single patches and multi-commit patch files (like those from GitHub pull request `.patch` URLs). An optional second parameter `cacheKeyPrefix` can be provided to generate cache keys for each file in the patch (format: `prefix-patchIndex-fileIndex`), enabling [caching of rendered diff results](https://diffs.com/docs#worker-pool-render-cache) in the worker pool.

An optional `throwOnError` parameter (default: `false`) controls error handling. When `true`, parsing errors throw exceptions; when `false`, errors are logged to the console and parsing continues on a best-effort basis.

### trimPatchContext

Trim patches with large context windows down to a fixed context window while keeping valid diff headers.

### preloadHighlighter

Preload specific themes and languages before rendering to ensure instant highlighting with no async loading delay.

Register a custom Shiki theme for use with any component. The theme name you register must match the `name` field inside your theme JSON file.

Register a custom Shiki language loader and optionally map it to file names or extensions. Use this when you're working with languages not bundled by Shiki or want custom highlighting grammars.

### setLanguageOverride

Override the syntax highlighting language for a `FileContents` or `FileDiffMetadata` object. This is useful when the filename doesn't have an extension or doesn't match the actual language.

## Styling

Diff and code components are rendered using shadow DOM APIs, allowing styles to be well-isolated from your page's existing CSS. However, it also means you may have to utilize some custom CSS variables to override default styles. These can be done in your global CSS, as style props on parent components, or on the `FileDiff` component directly.

### Advanced: Unsafe CSS

For advanced customization, you can inject arbitrary CSS into the shadow DOM using the `unsafeCSS` option. This CSS will be wrapped in an `@layer unsafe` block, giving it the highest priority in the cascade. Use this sparingly and with caution, as it bypasses the normal style isolation.

We also recommend that any CSS you apply uses simple, direct selectors targeting the existing data attributes. Avoid structural selectors like `:first-child`, `:last-child`, `:nth-child()`, sibling combinators (`+` or `~`), deeply nested descendant selectors, or bare tag selectors—these are susceptible to breaking in future versions or in edge cases that may be difficult to anticipate.

We cannot currently guarantee backwards compatibility for this feature across any future changes to the library, even in patch versions. Please reach out so that we can discuss a more permanent solution for modifying styles.

## Themes

Pierre Diffs ships with our custom open source themes, [Pierre Light and Pierre Dark](https://diffs.com/theme). We generate our themes with a custom build process that takes a shared color palette, assigns colors to specific roles for syntax highlighting, and builds JSON files and editor extensions. This makes our themes compatible with Shiki, Visual Studio Code, Cursor, and Zed.

| Editor / Platform | Source |
| --- | --- |
| Visual Studio Code | [VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=pierrecomputer.pierre-theme) |
| Cursor | [Open VSX](https://open-vsx.org/extension/pierrecomputer/pierre-theme) |
| Zed | [Zed Extensions](https://zed.dev/extensions/pierre-theme) |
| Shiki | [Theme repository](https://github.com/pierrecomputer/theme/tree/main/themes) |

While you can use any Shiki theme with Pierre Diffs by passing the theme name to the `theme` option, you can also create and register custom themes compatible with Shiki and Visual Studio Code. We recommend using our themes as a starting point for your own custom themes—head to our [themes documentation](https://diffs.com/theme) to get started.

[

Themes documentation

](https://diffs.com/theme)

## Token Hooks

Token hooks are experimental and subject to change.

Token hooks let you attach callbacks to syntax-highlighted tokens for custom hover UI, and LSP `textDocument/hover` integrations.

The shared prop tables in the React API and Vanilla JS API sections list the exact option names. This section covers behavior, examples, and performance tradeoffs.

Available on:

- React: `MultiFileDiff`, `PatchDiff`, `FileDiff`, and `File`
- Vanilla JS: `FileDiff` and `File`

Shared behavior:

- `onTokenEnter`, `onTokenLeave`, and `onTokenClick` receive `tokenText`, `lineNumber`, `lineCharStart`, `lineCharEnd`, and `tokenElement`. Diff variants also receive `side`.
- `lineCharStart` is zero-based and `lineCharEnd` is end-exclusive.
- If both token and line click handlers are attached, both will fire.
- Whitespace-only tokens are excluded unless `enableTokenInteractionsOnWhitespace` is `true`.
- `tokenElement` is usually the simplest way to apply temporary hover styles.
- Set `useTokenTransformer: true` when you want token wrappers or experimental selectors like `data-char` without token callbacks.
- Enabling token metadata increases DOM size because more token wrappers and attributes are preserved. On large files or many mounted diffs, this can have a noticeable performance cost.
- If you are using a [Worker Pool](https://diffs.com/docs#worker-pool), set `useTokenTransformer: true` on `WorkerPoolManager`. Worker pools can move highlighting work off the main thread, but they do not reduce the extra DOM size created by token metadata.
- If you are using [SSR](https://diffs.com/docs#ssr) don't forget to set `useTokenTransformer: true` on your preload option configs

## Worker Pool

This feature is experimental and undergoing active development. There may be bugs and the API is subject to change.

Import worker utilities from `@pierre/diffs/worker`.

By default, syntax highlighting runs on the main thread using Shiki. If you're rendering large files or many diffs, this can cause a bottleneck on your JavaScript thread resulting in jank or unresponsiveness. To work around this, we've provided some APIs to run all syntax highlighting in worker threads. The main thread will still attempt to render plain text synchronously and then apply the syntax highlighting when we get a response from the worker threads.

Basic usage differs a bit depending on if you're using React or Vanilla JS APIs, so continue reading for more details.

### Setup

One unfortunate side effect of using Web Workers is that different bundlers and environments require slightly different approaches to create a Web Worker. You'll need to create a function that spawns a worker that's appropriate for your environment and bundler and then pass that function to our provided APIs.

Lets begin with the `workerFactory` function. We've provided some examples for common use cases below.

Only the Vite and NextJS examples have been tested by us. Additional examples were generated by AI. If any of them are incorrect, please let us know.

#### Vite

You may need to explicitly set the `worker.format` option in your [Vite Config](https://vite.dev/config/worker-options#worker-format) to `'es'`.

#### NextJS

Workers only work in client components. Ensure your function has the `'use   client'` directive if using App Router.

#### VS Code Webview Extension

VS Code webviews have special security restrictions that require a different approach. You'll need to configure both the extension side (to expose the worker file) and the webview side (to load it via blob URL).

**Extension side:** Add the worker directory to `localResourceRoots` in your `getWebviewOptions()`:

Create the worker URI in `_getHtmlForWebview()`. Note: use `worker-portable.js` instead of `worker.js` — the portable version is designed for environments where ES modules aren't supported in web workers.

Pass the URI to the webview via an inline script in your HTML:

Your Content Security Policy must include `worker-src` and `connect-src`:

**Webview side:** Declare the global type for the URI:

Fetch the worker code and create a blob URL:

Create the `workerFactory` function:

#### Webpack 5

#### esbuild

#### Rollup / Static Files

If your bundler doesn't have special worker support, build and serve the worker file statically:

#### Vanilla JS (No Bundler)

For projects without a bundler, host the worker file on your server and reference it directly:

### Usage

With your `workerFactory` function created, you can integrate it with our provided APIs. In React, you'll want to pass this `workerFactory` to a `<WorkerPoolContextProvider>` so all components can inherit the pool automatically. If you're using the Vanilla JS APIs, we provide a `getOrCreateWorkerPoolSingleton` helper that ensures a single pool instance that you can then manually pass to all your File/FileDiff instances.

When using the worker pool, the `theme`, `lineDiffType`, `tokenizeMaxLineLength`, and `useTokenTransformer` render options are controlled by `WorkerPoolManager`, not individual components. Passing these options into component instances will be ignored.

To change render options after `WorkerPoolManager` instantiates, call `setRenderOptions()`. Changing render options will force mounted components to re-render and clear the render cache.

If you need token callbacks or experimental token selectors such as `data-char`, enable `useTokenTransformer: true` on the worker pool itself. Worker pools can move highlighting work off the main thread, but they do not reduce the extra DOM size created by token metadata. For token callback behavior and performance tradeoffs, see [Token Hooks](https://diffs.com/docs#token-hooks).

If you need to control which Shiki engine is used, set `preferredHighlighter` when initializing the pool (`'shiki-js'` by default, `'shiki-wasm'` optional).

#### React

Wrap your component tree with `WorkerPoolContextProvider` from `@pierre/diffs/react`. All `FileDiff` and `File` components nested within will automatically use the worker pool for syntax highlighting.

The `WorkerPoolContextProvider` will automatically spin up or shut down the worker pool based on its react lifecycle. If you have multiple context providers, they will all share the same pool, and termination won't occur until all contexts are unmounted.

Workers only work in client components. Ensure your function has the `'use   client'` directive if using App Router.

To change themes or other render options dynamically, use the `useWorkerPool()` hook to access the pool manager and call `setRenderOptions()`.

#### Vanilla JS

Use `getOrCreateWorkerPoolSingleton` to spin up a singleton worker pool. Then pass that as the second argument to `File` and/or `FileDiff`. When you are done with the worker pool, you can use `terminateWorkerPoolSingleton` to free up resources.

To change themes or other render options dynamically, call `setRenderOptions(options)` on the pool instance.

### Render Cache

This is an experimental feature being validated in production use cases. The API is subject to change.

The worker pool can cache rendered AST results to avoid redundant highlighting work. When a file or diff has a `cacheKey`, subsequent requests with the same key will return cached results immediately instead of reprocessing through a worker. This works automatically for both React and Vanilla JS APIs.

Caching is enabled per-file/diff by setting a `cacheKey` property. Files and diffs without a `cacheKey` will not be cached. The cache also validates against render options — if options like theme or line diff type change, the cached result is skipped and re-rendered.

### API Reference

These methods are exposed for advanced use cases. In most scenarios, you should use the `WorkerPoolContextProvider` for React or pass the pool instance via the `workerPool` option for Vanilla JS rather than calling these methods directly.

### Architecture

The worker pool manages a configurable number of worker threads that each initialize their own Shiki highlighter instance. Tasks are distributed across available workers, with queuing when all workers are busy.

## SSR

Import SSR utilities from `@pierre/diffs/ssr`.

The SSR API allows you to pre-render file diffs on the server with syntax highlighting, then hydrate them on the client for full interactivity.

If you pass `prerenderedHTML`, `onPostRender` still fires on the client after hydration. On later renders, it also fires after DOM-committing updates, whether those updates are a full replacement or a partial update. This is useful for measuring, observing, or otherwise manipulating the mounted diff container or content of the diffs itself.

### Usage

Each preload function returns an object containing the original inputs plus a `prerenderedHTML` string. This object can be spread directly into the corresponding React component for automatic hydration.

Inputs used for pre-rendering must exactly match what's rendered in the client component. We recommend spreading the entire result object into your File or Diff component to ensure the client receives the same inputs that were used to generate the pre-rendered HTML.

#### Server Component

#### Client Component

### Preloaders

We provide several preload functions to handle different input formats. Choose the one that matches your data source.

#### preloadFile

Preloads a single file with syntax highlighting (no diff). Use this when you want to render a file without any diff context. Spread into the `File` component.

#### preloadUnresolvedFile

Preloads a merge-conflict file for `UnresolvedFile` hydration. Use this when the file contains conflict markers (`<<<<<<<`, `=======`, `>>>>>>>`) and you want to preserve the unresolved conflict UI from SSR to client.

> **Experimental:** `UnresolvedFile` and `preloadUnresolvedFile` are currently beta/experimental and may change in future releases.

#### preloadFileDiff

Preloads a diff from a `FileDiffMetadata` object. Use this when you already have parsed diff metadata (e.g., from `parseDiffFromFile` or `parsePatchFiles`). Spread into the `FileDiff` component.

#### preloadMultiFileDiff

Preloads a diff directly from old and new file contents. This is the simplest option when you have the raw file contents and want to generate a diff. Spread into the `MultiFileDiff` component.

#### preloadPatchDiff

Preloads a diff from a unified patch string for a single file. Use this when you have a patch in unified diff format. Spread into the `PatchDiff` component.

#### preloadPatchFile

Preloads multiple diffs from a multi-file patch string. Returns an array of results, one for each file in the patch. Each result can be spread into a `FileDiff` component.