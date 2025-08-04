---
allowed-tools: Task, Read, Edit, MultiEdit, Grep, Glob
description: Update documentation for modified features
argument-hint: [specific feature or leave empty for full sync]
---
# 📚 Documentation Synchronization

I need to update documentation to reflect recent code changes.

**Use the documentation-maintainer agent** to:

1. **Detect documentation drift**:
   - Identify modified features: ${ARGUMENTS:-all}
   - Check for outdated examples
   - Find missing documentation
   - Verify API documentation accuracy
   - Review README completeness

2. **Update documentation files**:
   - Sync README.md with current features
   - Update example code in /examples
   - Refresh API documentation
   - Update configuration references
   - Fix broken links

3. **Maintain consistency**:
   - Ensure godoc comments match implementation
   - Update CHANGELOG entries
   - Sync command-line help text
   - Verify all code examples compile
   - Update performance metrics

4. **Documentation areas**:
   - Installation instructions
   - Configuration options
   - Usage examples
   - API reference
   - Troubleshooting guides
   - Performance documentation

5. **Quality checks**:
   - Spelling and grammar
   - Code example validation
   - Link verification
   - Version number consistency
   - Screenshot updates if needed

The agent will ensure all documentation accurately reflects the current state of the codebase with working examples and clear explanations.
