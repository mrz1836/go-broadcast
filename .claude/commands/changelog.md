---
allowed-tools: Task, Bash(git log:*), Read, Edit, MultiEdit
description: Generate changelog from commits
argument-hint: [version tag or commit range]
---
!git log --oneline -10 --pretty=format:"%h %s"

# 📝 Changelog Generation

I need to generate a comprehensive changelog from recent commits.

**Use the changelog-generator agent** to:

1. **Analyze commits**:
   - Parse commit range: ${ARGUMENTS:-since last release}
   - Identify conventional commit types
   - Group by feature/fix/chore categories
   - Extract breaking changes
   - Find notable improvements

2. **Categorize changes**:
   - ✨ **Features**: New functionality
   - 🐛 **Bug Fixes**: Resolved issues
   - 🚀 **Performance**: Speed improvements
   - 🔒 **Security**: Security fixes
   - 📚 **Documentation**: Doc updates
   - 🔧 **Maintenance**: Internal changes
   - 💥 **Breaking Changes**: API changes

3. **Generate entries**:
   - Follow conventional changelog format
   - Include PR numbers and authors
   - Link to relevant issues
   - Highlight important changes
   - Add migration notes for breaking changes

4. **Format output**:
   - Markdown formatted changelog
   - Version header with date
   - Grouped by change type
   - Clear, concise descriptions
   - Compare links to previous versions

5. **Integration**:
   - Update CHANGELOG.md
   - Prepare release notes
   - Generate PR/issue summaries
   - Create version announcements

The agent will produce a professional changelog ready for releases and user communication.
