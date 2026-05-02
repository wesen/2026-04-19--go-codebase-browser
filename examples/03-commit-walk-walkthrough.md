# Commit Walk: Review the Static Export Pipeline

A step-by-step walkthrough of the commits affecting the static export pipeline.

```codebase-commit-walk from=HEAD~5 to=HEAD
step kind=overview title="Review scope" body="This walkthrough covers commits touching the static export pipeline."
step kind=diff-stats title="Change summary"
step kind=symbol sym=staticapp.Export title="Inspect the Export function"
step kind=diff sym=staticapp.Export from=HEAD~5 to=HEAD title="Diff across recent changes"
step kind=impact sym=staticapp.Export dir=usedby depth=2 title="Callers"
step kind=note title="Key observation" body="The new Options field was added to make --include-source configurable without changing the function signature."
```
