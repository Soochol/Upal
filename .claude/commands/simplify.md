Simplify and refine recently modified code for clarity, consistency, and maintainability.

Steps:
1. Check for uncommitted changes (`git status --porcelain`). If any exist, commit them first using the /commit skill
2. Identify the HEAD commit hash from the git status shown at conversation start. Run `git diff --name-only <starting-commit>..HEAD | sort -u` to get only files changed during this session
3. Use the Task tool with `subagent_type: "code-simplifier:code-simplifier"` to launch the code-simplifier agent
4. Pass the list of modified files and ask it to review and simplify them while preserving all functionality
5. Report what was changed to the user
