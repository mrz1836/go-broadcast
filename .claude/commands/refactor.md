---
allowed-tools: Task, Read, Edit, MultiEdit, Grep
description: Refactor code for better structure and maintainability
argument-hint: @<file-or-component-path>
---
# 🔧 Code Refactoring

I need to refactor the specified code: $ARGUMENTS

**Use the refactoring-specialist agent** to:

1. **Code analysis**:
   - Read and understand the target code
   - Identify improvement opportunities
   - Analyze current design patterns
   - Check complexity metrics
   - Review error handling

2. **Refactoring targets**:
   - **Structure**: Improve code organization
   - **Interfaces**: Design better abstractions
   - **Functions**: Reduce complexity, improve names
   - **Error handling**: Implement consistent patterns
   - **Types**: Improve type safety
   - **Tests**: Enhance test coverage

3. **Go best practices**:
   - Apply idiomatic Go patterns
   - Use composition over inheritance
   - Implement proper interfaces
   - Follow single responsibility
   - Apply dependency injection
   - Use context appropriately

4. **Specific improvements**:
   - Extract complex logic into functions
   - Reduce cyclomatic complexity
   - Improve variable/function names
   - Add missing error checks
   - Implement proper logging
   - Optimize performance hotspots

5. **Quality checks**:
   - Ensure tests still pass
   - Maintain backwards compatibility
   - Improve code coverage
   - Update documentation
   - Run linters post-refactor

The agent will transform the code into a cleaner, more maintainable version while preserving all functionality.
