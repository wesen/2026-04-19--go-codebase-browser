# Changelog

## 2026-04-25

- Initial workspace created


## 2026-04-25

Created GCB-011 to adopt @pierre/diffs for nicer diff rendering; downloaded https://diffs.com/docs with Defuddle into sources/; added initial design plan, diary, tasks, and related file links.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/04/25/GCB-011--adopt-diffs-library-for-nicer-semantic-diff-rendering/design/01-diffs-library-adoption-plan-for-semantic-diff-widgets.md — Initial adoption plan
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/04/25/GCB-011--adopt-diffs-library-for-nicer-semantic-diff-rendering/reference/01-investigation-diary.md — Initial diary
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ttmp/2026/04/25/GCB-011--adopt-diffs-library-for-nicer-semantic-diff-rendering/sources/01-diffs-docs.md — Defuddle-exported Diffs documentation


## 2026-04-25

Implemented the first @pierre/diffs integration: added dependency, shared MultiFileDiff wrapper, unified/split toggle, word-level diff mode, migrated embedded and history symbol diffs, and redesigned codebase-annotation into a clearer review-note widget.

### Related Files

- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/diff/DiffsUnifiedDiff.tsx — Shared Diffs wrapper with unified/split toggle and word-level diffs
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/widgets/AnnotationWidget.tsx — Redesigned as clear review-note widget
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/doc/widgets/SymbolDiffInlineWidget.tsx — Uses shared Diffs wrapper
- /home/manuel/code/wesen/2026-04-19--go-codebase-browser/ui/src/features/history/HistoryPage.tsx — Focused symbol body diff uses shared Diffs wrapper

