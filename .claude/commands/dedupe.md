---
allowed-tools: Task, Grep, Read, Edit, MultiEdit
description: Find and remove duplicate code patterns
argument-hint: [minimum lines threshold or leave empty for default]
---
# 🔍 Code Deduplication Analysis

I need to find and eliminate duplicate code patterns.

**Sequential deduplication workflow:**

1. **Use the code-deduplicator agent** to:
   - Scan codebase for duplicate patterns
   - Identify copy-paste code
   - Find similar function implementations
   - Detect repeated logic blocks
   - Analyze structural duplicates
   - Set threshold: ${ARGUMENTS:-15 lines}

2. **Then use the refactoring-specialist agent** to:
   - Extract common code into reusable functions
   - Create shared utilities
   - Implement proper interfaces
   - Design generic solutions
   - Apply DRY principles

**Analysis targets:**
- Exact code duplicates
- Similar function patterns
- Repeated error handling
- Duplicate test helpers
- Common validation logic
- Repeated struct definitions

**Refactoring approach:**
- Extract shared functions
- Create utility packages
- Implement generic types
- Use composition patterns
- Apply template methods
- Create factory functions

**Quality assurance:**
- Maintain test coverage
- Preserve functionality
- Improve maintainability
- Document extracted code
- Update all references

The agents will identify all duplicate code and refactor it into clean, reusable components.
