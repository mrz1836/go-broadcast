package cli

// TestValidConfig is a common valid configuration used in tests
const TestValidConfig = `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: org/template
      branch: main
    targets:
      - repo: org/target1
        files:
          - src: README.md
            dest: README.md`
