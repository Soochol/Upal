# Upal Landing Page Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a dark-first landing page for Upal inspired by Google Opal / modern AI SaaS tools, with 8 sections: Hero, Providers, Features Grid, How It Works, Product Showcase, Use Cases, Final CTA, Footer.

**Architecture:** Add React Router to the SPA. `/` serves the landing page, `/editor` serves the existing workflow editor. The landing page is a single `Landing.tsx` page component using the existing Tailwind CSS + dark theme variables. The Go backend's `StaticHandler` already does SPA fallback so no backend changes needed.

**Tech Stack:** React 19, React Router v7, Tailwind CSS v4, Lucide icons, existing design tokens (oklch color system)

---

## Task 1: Install React Router and set up routing

**Files:**
- Modify: `web/package.json`
- Modify: `web/src/main.tsx`
- Modify: `web/src/App.tsx`
- Create: `web/src/pages/Landing.tsx` (placeholder)
- Create: `web/src/pages/Editor.tsx` (extract from App.tsx)

**Step 1: Install react-router-dom**

Run: `cd /home/dev/code/Upal/web && npm install react-router-dom`

**Step 2: Create `pages/Editor.tsx`**

Move the entire current `App.tsx` content into `pages/Editor.tsx`, renaming the component to `Editor`.

```tsx
// web/src/pages/Editor.tsx
// (exact copy of current App.tsx content, renamed function App → function Editor, export default Editor)
```

**Step 3: Create placeholder `pages/Landing.tsx`**

```tsx
// web/src/pages/Landing.tsx
export default function Landing() {
  return <div className="dark bg-background text-foreground min-h-screen">Landing Page</div>
}
```

**Step 4: Update `main.tsx` to use React Router**

```tsx
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import './index.css'
import Landing from './pages/Landing'
import Editor from './pages/Editor'
import { ThemeProvider } from './components/ThemeProvider'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeProvider defaultTheme="light" storageKey="upal-ui-theme">
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Landing />} />
          <Route path="/editor" element={<Editor />} />
        </Routes>
      </BrowserRouter>
    </ThemeProvider>
  </StrictMode>,
)
```

**Step 5: Slim down `App.tsx`**

App.tsx can simply re-export or redirect. Or remove it entirely if unused.

**Step 6: Verify routing works**

Run: `cd /home/dev/code/Upal/web && npm run build`
Expected: Build succeeds with no errors.

**Step 7: Commit**

```bash
git add web/package.json web/package-lock.json web/src/main.tsx web/src/pages/
git commit -m "feat(web): add React Router, split editor into /editor route"
```

---

## Task 2: Build Landing Page — Hero Section

**Files:**
- Modify: `web/src/pages/Landing.tsx`
- Modify: `web/src/index.css` (add landing-specific styles)

**Implementation:**

The Hero section is the most important. Dark navy background (using `.dark` class), large headline, subheadline, two CTA buttons, and a product screenshot with glow border.

Key design elements:
- Force `dark` class on the landing page wrapper
- Hero takes full viewport height
- Headline: "Build AI Workflows, Visually." (56-72px, font-bold)
- Subheadline: "Connect multiple AI models into powerful pipelines — drag, drop, and run."
- Two CTAs: "Start Building" (primary, links to /editor) and "GitHub" (outline, external)
- Product preview: bordered screenshot image with purple glow shadow
- Sticky navigation bar at top: Logo, Features, How it Works, GitHub link, "Get Started" button
- Subtle gradient background: deep navy to slightly lighter navy

**CSS additions in index.css:**
- `.landing-glow` — purple glow box-shadow for hero image
- `@keyframes float` — subtle float animation for hero image
- `@keyframes fade-in-up` — scroll reveal animation

**Step 1: Add landing-specific CSS to index.css**
**Step 2: Build the full Hero section in Landing.tsx with nav**
**Step 3: Verify visually**

Run: `cd /home/dev/code/Upal/web && npm run dev`
Check http://localhost:5173/ shows the hero section.

**Step 4: Commit**

---

## Task 3: Build Landing Page — Providers Bar + Features Grid

**Files:**
- Modify: `web/src/pages/Landing.tsx`

**Provider bar:** Horizontal row of LLM provider names/logos with "Works with" label above. Use text-based logos with icons since we don't have SVG logos: Anthropic, Google Gemini, OpenAI, Ollama.

**Features Grid (3x2):**

| Feature | Icon | Description |
|---------|------|-------------|
| Visual DAG Editor | `GitBranch` | Drag and drop nodes to build AI workflows on an interactive canvas |
| Multi-Model Support | `Cpu` | Connect Anthropic, Gemini, OpenAI, and Ollama models in one pipeline |
| Natural Language Gen | `Sparkles` | Describe your workflow in plain English and watch it build itself |
| Real-time Execution | `Activity` | Stream results live with SSE — see each node execute in real time |
| Tool Integration | `Wrench` | Connect external tools and APIs as workflow nodes |
| Agentic Loops | `RefreshCw` | Autonomous agents that iterate with tool calls until the task is done |

Each card: glassmorphism style (semi-transparent bg, border, subtle blur), icon in node-type color, title, 1-line description.

**Step 1: Build Providers section**
**Step 2: Build Features grid section**
**Step 3: Verify visually**
**Step 4: Commit**

---

## Task 4: Build Landing Page — How It Works (3-Step)

**Files:**
- Modify: `web/src/pages/Landing.tsx`

Three steps with connecting lines between them:
1. **Design** — "Drag nodes onto the canvas and configure your AI models" (node-input color)
2. **Connect** — "Link nodes with edges to define your data flow pipeline" (node-agent color)
3. **Run** — "Execute with one click and watch results stream in real-time" (node-output color)

Each step: large number, colored icon, title, description. Connected by dashed lines (similar to edge style in editor).

**Step 1: Build How It Works section**
**Step 2: Verify visually**
**Step 3: Commit**

---

## Task 5: Build Landing Page — Product Showcase

**Files:**
- Modify: `web/src/pages/Landing.tsx`

Large, centered screenshot of the editor (dark mode). Use the existing `final-dark-navy.png` as the showcase image. Copy it to `web/public/screenshots/editor-dark.png`.

Framed in a browser-like window chrome (rounded corners, dots at top-left). Purple/blue gradient border glow.

**Step 1: Copy screenshot to web/public**
**Step 2: Build Product Showcase section**
**Step 3: Verify visually**
**Step 4: Commit**

---

## Task 6: Build Landing Page — Use Cases + Final CTA + Footer

**Files:**
- Modify: `web/src/pages/Landing.tsx`

**Use Cases (3 cards):**
1. "AI Chatbot Pipeline" — Build conversational AI with context, tools, and memory
2. "Content Generation" — Chain prompts to research, draft, and refine content automatically
3. "Data Analysis" — Process data through multiple AI models for insights and summaries

Each card: glassmorphism, icon, title, description.

**Final CTA:**
- Dark section with centered text
- "Ready to build your AI workflow?"
- "Start Building" button (primary, large)

**Footer:**
- Simple row: Upal logo + copyright, GitHub link, "Built with Go + React"
- Muted colors, minimal

**Step 1: Build Use Cases section**
**Step 2: Build Final CTA section**
**Step 3: Build Footer**
**Step 4: Verify full page scroll**
**Step 5: Commit**

---

## Task 7: Polish — Scroll Animations + Responsive

**Files:**
- Modify: `web/src/pages/Landing.tsx`
- Modify: `web/src/index.css`

Add CSS-only scroll animations using `@keyframes` and intersection observer (or CSS `animation-timeline: view()` if supported).

Responsive breakpoints:
- Mobile (<640px): Stack everything vertically, single column features
- Tablet (640-1024px): 2-column features grid
- Desktop (>1024px): 3-column features grid, full layout

**Step 1: Add scroll-triggered fade-in animations**
**Step 2: Ensure responsive layout at all breakpoints**
**Step 3: Final visual QA**
**Step 4: Run build to verify no errors**

Run: `cd /home/dev/code/Upal/web && npm run build`
Expected: Build succeeds.

**Step 5: Commit**

```bash
git add -A
git commit -m "feat(web): complete landing page with animations and responsive design"
```
