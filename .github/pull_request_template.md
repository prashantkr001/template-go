Please include a summary of the change, relevant motivation, and context.

---

- [ ] PR title must start with an exact match of: `chore: `, `fix: `, or `feat: `

* _chore_: no version bump
* _fix_: patch version bump, non-breaking change which fixes an issue
* _feat_: minor version bump, new feature, non-breaking change which adds functionality

To trigger a major version bump, you must indicate a breaking change. This is done by adding `BREAKING CHANGE:` in the footer of the commit message or by adding a `!` after the type/scope. This is for changes that are not backward-compatible.

e.g.

```
feat!: breaking feature introduced, will bump major version
```
