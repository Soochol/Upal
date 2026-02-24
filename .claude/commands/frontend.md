Start (or restart) the frontend vite dev server.

Steps:
1. Check if a process is already running on port 5173: `lsof -ti :5173`
2. If a process is found, kill it: `kill <pid>`
3. Start the frontend in the background: run `cd web && npm run dev` using `Bash` with `run_in_background: true`
4. Wait a moment, then verify it started by checking port 5173
5. Report the status to the user
