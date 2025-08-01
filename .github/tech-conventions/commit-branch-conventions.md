# Commit & Branch Naming Conventions

> Clear history ⇒ easy maintenance. Follow these rules for every commit and branch.

<br><br>

## 📌 Commit Message Format

```
<type>(<scope>): <imperative short description>

<body>  # optional, wrap at 72 chars
```

* **`<type>`** — `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `build`, `ci`
* **`<scope>`** — Affected subsystem or package (e.g., `api`, `deps`). Omit if global.
* **Short description** — ≤ 50 chars, imperative mood ("add pagination", "fix panic")
* **Body** (optional) — What & why, links to issues (`Closes #123`), and breaking‑change note (`BREAKING CHANGE:`)

**Examples**

```
feat(package): add new method called: Thing()
fix(generator): handle malformed JSON input gracefully
docs(README): improve installation instructions
```

> Commits that only tweak whitespace, comments, or docs inside a PR may be squashed; otherwise preserve granular commits.

<br><br>

## 📝 Pre-Commit Hooks (Optional)

To ensure consistent commit messages, we use a pre-commit hook that checks the format before allowing a commit. The hook is defined in `.pre-commit-config.yaml` and can be installed with:

```bash
pre-commit install
```

If you don't have `pre-commit` installed, you can install it via Homebrew:
```bash
brew install pre-commit
```

Run the pre-commit hook manually with:
```bash
pre-commit run --all-files
```

> The pre-commit hook will automatically check your commit messages against the defined format and prevent commits that do not comply.

<br><br>

## 🌱 Branch Naming

| Purpose            | Prefix      | Example                            |
|--------------------|-------------|------------------------------------|
| Bug Fix            | `fix/`      | `fix/code-off-by-one`              |
| Chore / Meta       | `chore/`    | `chore/upgrade-go-1.24`            |
| Documentation      | `docs/`     | `docs/agents-commenting-standards` |
| Feature            | `feat/`     | `feat/pagination-api`              |
| Hotfix (prod)      | `hotfix/`   | `hotfix/rollback-broken-deploy`    |
| Prototype / Spike  | `proto/`    | `proto/iso3166-expansion`          |
| Refactor / Cleanup | `refactor/` | `refactor/remove-dead-code`        |
| Tests              | `test/`     | `test/generator-edge-cases`        |

* Use **kebab‑case** after the prefix.
* Keep branch names concise yet descriptive.
* PR titles should mirror the branch's purpose (see Pull Request Guidelines).

> CI rely on these prefixes for auto labeling and workflow routing—stick to them.

<br><br>

## 🎯 Commit Best Practices

### Do:
* **Write atomic commits** — Each commit should represent one logical change
* **Use imperative mood** — "Add feature" not "Added feature"
* **Reference issues** — Include issue numbers when applicable
* **Explain why** — Use the commit body for context
* **Sign your commits** — Use `git commit -s` when required

### Don't:
* **Mix unrelated changes** — Keep commits focused
* **Commit broken code** — Every commit should be buildable
* **Use generic messages** — "Fix bug" or "Update code" are too vague
* **Forget to proofread** — Check spelling and grammar

<br><br>

## 📊 Commit Message Examples

### Good Examples

```
feat(auth): add JWT token refresh endpoint

Implements automatic token refresh to improve user experience.
Tokens are refreshed 5 minutes before expiration.

Closes #234
```

```
fix(worker): prevent goroutine leak in batch processor

The worker pool was not properly cleaning up goroutines when
context was cancelled, leading to memory leaks in long-running
services.

Added proper context handling and wait group synchronization.
```

```
refactor(cache): simplify TTL calculation logic

Extracted TTL calculation into separate function to improve
testability and reduce cognitive complexity.

No functional changes.
```

### Bad Examples

```
fixed stuff          # Too vague
WIP                  # Meaningless
Update auth.go       # What was updated and why?
Bug fix              # Which bug? What was the issue?
```

<br><br>

## 🔄 Working with Git

### Useful Commands

```bash
# Amend the last commit (before pushing)
git commit --amend

# Interactive rebase to clean up history (use with caution)
git rebase -i HEAD~3

# Show commit history with graph
git log --oneline --graph --all

# Cherry-pick specific commits
git cherry-pick <commit-hash>
```

### Branch Management

```bash
# Create and switch to new branch
git checkout -b feat/new-feature

# Delete local branch
git branch -d feat/old-feature

# Delete remote branch
git push origin --delete feat/old-feature

# Prune deleted remote branches
git remote prune origin
```

> Always pull with rebase to keep history clean: `git pull --rebase`
