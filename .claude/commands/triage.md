---
allowed-tools: Task, Bash(gh issue list:*), Bash(gh issue edit:*), Read
description: Manage and organize GitHub issues
argument-hint: [new|stale|unlabeled|all]
---
!gh issue list --limit 10 --json number,title,state,labels | head -20

# 📋 Issue Triage Management

I need to triage and organize GitHub issues: ${ARGUMENTS:-all}

**Use the issue-triage-bot agent** to:

1. **New issue triage**:
   - Categorize by type (bug/feature/question)
   - Apply appropriate labels
   - Set priority levels
   - Assign to correct milestone
   - Add to relevant project boards
   - Notify appropriate team members

2. **Stale issue management**:
   - Identify inactive issues (>30 days)
   - Add stale labels with warnings
   - Close abandoned issues (>60 days)
   - Archive resolved discussions
   - Update reporter on status

3. **Label organization**:
   - Ensure consistent labeling
   - Apply multiple relevant labels
   - Use label hierarchy properly
   - Add missing categorization
   - Remove outdated labels

4. **Duplicate detection**:
   - Find similar issues
   - Link related problems
   - Consolidate discussions
   - Close duplicates appropriately
   - Preserve important context

5. **Assignment logic**:
   - Route to subject experts
   - Balance workload
   - Consider availability
   - Match skills to issues
   - Set realistic timelines

**Triage categories:**
- 🐛 Bugs by severity
- ✨ Feature requests by impact
- 📚 Documentation needs
- 🔧 Maintenance tasks
- ❓ Questions requiring answers

The agent will ensure all issues are properly organized for efficient resolution.
