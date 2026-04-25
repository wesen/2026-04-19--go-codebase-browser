# Slice 5 Demo: Guided commit walk

This page demonstrates `codebase-commit-walk`: a guided multi-step review widget
that composes the smaller semantic widgets into one literate review path.

```codebase-commit-walk title="Slice 0 commit-aware snippets review"
# Step lines support quoted title/body values and the same params as the widgets they compose.
step kind=stats title="Start with the overall size" body="This change is small enough to review semantically instead of file-by-file." from=c9132579687ca9b334ff81f9161ba058bffc52c4 to=e457069b75f87ddc0a07a58d5608e96334a7fcf0
step kind=files title="Check which files moved" body="The review surface is server and docs rendering code; the file list confirms the scope." from=c9132579687ca9b334ff81f9161ba058bffc52c4 to=e457069b75f87ddc0a07a58d5608e96334a7fcf0
step kind=diff title="Inspect handleSnippet" body="This symbol gained the commit-aware branch that routes historical snippets through the history DB." sym=sym:github.com/wesen/codebase-browser/internal/server.method.Server.handleSnippet from=c9132579687ca9b334ff81f9161ba058bffc52c4 to=e457069b75f87ddc0a07a58d5608e96334a7fcf0
step kind=annotation title="Zoom in on the new branch" body="The highlighted lines are the important control-flow handoff." sym=sym:github.com/wesen/codebase-browser/internal/server.method.Server.handleSnippet commit=e457069b75f87ddc0a07a58d5608e96334a7fcf0 lines=9-13 note="Commit-aware snippets use the history DB when a commit is present."
step kind=history title="Check historical stability" body="The symbol history shows when the body hash changed and lets reviewers choose other from/to pairs." sym=sym:github.com/wesen/codebase-browser/internal/server.method.Server.handleSnippet limit=5
step kind=impact title="Finish with impact" body="Impact keeps the review connected to callers instead of treating the diff as isolated text." sym=sym:github.com/wesen/codebase-browser/internal/server.method.Server.handleSnippet dir=usedby depth=1 commit=e457069b75f87ddc0a07a58d5608e96334a7fcf0
```
