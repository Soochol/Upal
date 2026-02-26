AI/LLM이 생성한 코드를 체계적으로 리뷰하여 바이브 코딩에서 흔히 발생하는 아키텍처 문제를 찾아낸다.

Steps:
1. Check for uncommitted changes (`git status --porcelain`). If any exist, commit them first using the /commit skill
2. Identify the HEAD commit hash from the git status shown at conversation start. Run `git diff --name-only <starting-commit>..HEAD | sort -u` to get only files changed during this session
3. Use the Skill tool to invoke the `vibe-code-review` skill
4. Pass the list of modified files for review
5. Report findings to the user with actionable suggestions
