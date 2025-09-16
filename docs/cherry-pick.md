# Cherry-pick Guide for Tekton Operator

This document explains how to cherry-pick commits from the `main` branch to release branches like `release-v0.77.x`.

## Automated Cherry-pick

We have two automated cherry-pick workflows available:

### Method 1: Comment-based Cherry-pick (Recommended)

After a PR is merged to `main`, you can cherry-pick it to a release branch by commenting on the PR:

```
/cherry-pick release-v0.77.x
```

**Requirements:**
- You must be a repository member, owner, or collaborator
- The PR must be already merged
- The target branch must exist

**What happens:**
1. The bot extracts the merge commit from the PR
2. Creates a new branch based on the target release branch
3. Attempts to cherry-pick the commit
4. If successful: Creates a new PR to the release branch
5. If conflicts: Comments with manual instructions

### Method 2: Label-based Cherry-pick

Add a label to your PR before merging:

```
cherry-pick/release-v0.77.x
```

This will automatically attempt to cherry-pick the PR when it's merged to `main`.

## Manual Cherry-pick

If automated cherry-pick fails due to conflicts, follow these steps:

### 1. Checkout the release branch

```bash
git fetch origin release-v0.77.x
git checkout -b cherry-pick-fix-to-release-v0.77.x origin/release-v0.77.x
```

### 2. Cherry-pick the commit

```bash
# Get the commit SHA from the merged PR
git cherry-pick <commit-sha>
```

### 3. Resolve conflicts (if any)

If there are conflicts:

```bash
# Edit the conflicted files
git add .
git cherry-pick --continue
```

### 4. Push and create PR

```bash
git push origin cherry-pick-fix-to-release-v0.77.x

# Create PR using GitHub CLI
gh pr create \
  --base release-v0.77.x \
  --head cherry-pick-fix-to-release-v0.77.x \
  --title "[release-v0.77.x] Your PR Title" \
  --body "Cherry-pick of #<pr-number> to release-v0.77.x"
```

## Common Issues and Solutions

### Issue 1: "Branch does not exist"

**Problem:** The target release branch doesn't exist yet.

**Solution:** Create the release branch first:

```bash
# Create release branch from main
git checkout main
git pull origin main
git checkout -b release-v0.77.x
git push origin release-v0.77.x
```

### Issue 2: "Cherry-pick conflicts"

**Problem:** The commit cannot be applied cleanly due to conflicts.

**Solutions:**
1. **Resolve manually** (see manual cherry-pick steps above)
2. **Skip if not applicable** - Some changes might not be needed in the release branch
3. **Create a custom fix** - Sometimes a different approach is needed for the release branch

### Issue 3: "Merge commit cherry-pick fails"

**Problem:** Trying to cherry-pick a merge commit without specifying parent.

**Solution:** Specify the parent when cherry-picking:

```bash
# Use -m 1 for first parent (usually main branch changes)
git cherry-pick -m 1 <merge-commit-sha>
```

### Issue 4: "Permission denied"

**Problem:** The bot doesn't have permission to create PRs or push branches.

**Solution:** Ensure the GitHub token has the required permissions:
- `contents: write`
- `pull-requests: write`
- `issues: write`

## Best Practices

1. **Test before cherry-picking**: Ensure the change works in the release context
2. **Cherry-pick promptly**: Don't wait too long after merging to main
3. **Use descriptive branch names**: Include the target branch and purpose
4. **Update release notes**: Document cherry-picked changes in release notes
5. **Verify CI passes**: Ensure all tests pass on the cherry-pick PR

## Release Branch Naming Convention

- `release-v0.77.x` - for v0.77.x patch releases
- `release-v0.78.x` - for v0.78.x patch releases
- etc.

## Getting Help

If you encounter issues with cherry-picking:

1. Check this documentation
2. Look for similar issues in the repository
3. Ask in the `#tekton-operator` Slack channel
4. Create an issue with the `cherry-pick` label

