Start (or restart) the backend dev server using air (hot-reload).

Steps:
1. Check if a process is already running on port 8081: `lsof -ti :8081`
2. If a process is found, kill it: `kill <pid>`
3. Start the backend in the background: run `air` from the project root using `Bash` with `run_in_background: true`
4. Wait a moment, then verify it started by checking port 8081
5. Report the status to the user
