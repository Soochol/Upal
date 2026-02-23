/**
 * Decorative SVG background for the Preview panel idle state.
 * Renders a schematic wireframe of a generic app — suggesting what the
 * workflow would produce when run. Uses `currentColor` throughout so it
 * adapts to light / dark mode automatically.
 */
export function PreviewBackground() {
  return (
    <svg
      viewBox="0 0 360 480"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      className="absolute inset-0 w-full h-full pointer-events-none select-none text-foreground"
      preserveAspectRatio="xMidYMid meet"
      aria-hidden
    >
      <defs>
        {/* Fine dot grid */}
        <pattern id="pbg-dots" x="0" y="0" width="16" height="16" patternUnits="userSpaceOnUse">
          <circle cx="8" cy="8" r="0.85" fill="currentColor" opacity="0.22" />
        </pattern>

        {/* Clip the window interior to its rounded rect */}
        <clipPath id="pbg-clip">
          <rect x="64" y="52" width="232" height="368" rx="16" />
        </clipPath>
      </defs>

      {/* ── Dot grid ──────────────────────────────── */}
      <rect width="360" height="480" fill="url(#pbg-dots)" />

      {/* ── Engineering corner brackets ───────────── */}
      <g stroke="currentColor" strokeWidth="1.5" opacity="0.18" strokeLinecap="round">
        <path d="M28 48 L28 28 L48 28" />
        <path d="M332 28 L312 28 M332 28 L332 48" />
        <path d="M28 432 L28 452 L48 452" />
        <path d="M332 432 L332 452 L312 452" />
      </g>

      {/* ── Cross-hair registration marks at window corners ── */}
      <g stroke="currentColor" strokeWidth="0.75" opacity="0.15" strokeLinecap="round">
        {/* TL */}
        <line x1="60" y1="52" x2="68" y2="52" /><line x1="64" y1="48" x2="64" y2="56" />
        {/* TR */}
        <line x1="292" y1="52" x2="300" y2="52" /><line x1="296" y1="48" x2="296" y2="56" />
        {/* BL */}
        <line x1="60" y1="420" x2="68" y2="420" /><line x1="64" y1="416" x2="64" y2="424" />
        {/* BR */}
        <line x1="292" y1="420" x2="300" y2="420" /><line x1="296" y1="416" x2="296" y2="424" />
      </g>

      {/* ── Large dashed orbit ring ────────────────── */}
      <circle cx="180" cy="236" r="168" stroke="currentColor" strokeWidth="0.5"
        strokeDasharray="3 9" opacity="0.06" />

      {/* ── App window drop-shadow ─────────────────── */}
      <rect x="67" y="55" width="232" height="368" rx="16" fill="currentColor" opacity="0.04" />

      {/* ── Clipped window interior ───────────────── */}
      <g clipPath="url(#pbg-clip)">

        {/* Chrome bar fill */}
        <rect x="64" y="52" width="232" height="44" fill="currentColor" opacity="0.06" />
        {/* Chrome bar separator */}
        <line x1="64" y1="96" x2="296" y2="96" stroke="currentColor" strokeWidth="0.5" opacity="0.14" />

        {/* Traffic-light dots */}
        <circle cx="83" cy="74" r="5" fill="currentColor" opacity="0.22" />
        <circle cx="98" cy="74" r="5" fill="currentColor" opacity="0.22" />
        <circle cx="113" cy="74" r="5" fill="currentColor" opacity="0.22" />

        {/* Address / URL bar */}
        <rect x="132" y="65" width="112" height="18" rx="9" fill="currentColor" opacity="0.1" />
        <rect x="252" y="66" width="16" height="16" rx="4" fill="currentColor" opacity="0.1" />

        {/* Navigation bar */}
        <rect x="76" y="106" width="36" height="8" rx="4" fill="currentColor" opacity="0.24" />
        <rect x="76" y="118" width="36" height="2"  rx="1" fill="currentColor" opacity="0.22" />
        <rect x="120" y="106" width="28" height="8" rx="4" fill="currentColor" opacity="0.1" />
        <rect x="156" y="106" width="32" height="8" rx="4" fill="currentColor" opacity="0.1" />
        <rect x="196" y="106" width="26" height="8" rx="4" fill="currentColor" opacity="0.1" />
        {/* Nav avatar */}
        <circle cx="281" cy="110" r="11" fill="currentColor" opacity="0.1" />

        {/* Nav separator */}
        <line x1="64" y1="124" x2="296" y2="124" stroke="currentColor" strokeWidth="0.5" opacity="0.1" />

        {/* ── Hero section ── */}
        <rect x="76" y="138" width="116" height="13" rx="6.5" fill="currentColor" opacity="0.22" />
        <rect x="76" y="158" width="192" height="7"  rx="3.5" fill="currentColor" opacity="0.1"  />
        <rect x="76" y="170" width="156" height="7"  rx="3.5" fill="currentColor" opacity="0.08" />

        {/* CTA button */}
        <rect x="76" y="188" width="80" height="28" rx="8" fill="currentColor" opacity="0.14" />
        <rect x="164" y="190" width="48" height="24" rx="7" stroke="currentColor" strokeWidth="1" opacity="0.1" />

        {/* ── 2-column card grid ── */}
        {/* Card 1 */}
        <rect x="76" y="230" width="97"  height="82" rx="10" stroke="currentColor" strokeWidth="0.75" fill="currentColor" opacity="0.03" />
        <rect x="82" y="236" width="85"  height="46" rx="7" fill="currentColor" opacity="0.08" />
        <rect x="82" y="288" width="64"  height="8"  rx="4" fill="currentColor" opacity="0.18" />
        <rect x="82" y="301" width="48"  height="6"  rx="3" fill="currentColor" opacity="0.09" />

        {/* Card 2 */}
        <rect x="183" y="230" width="97" height="82" rx="10" stroke="currentColor" strokeWidth="0.75" fill="currentColor" opacity="0.03" />
        <rect x="189" y="236" width="85" height="46" rx="7" fill="currentColor" opacity="0.08" />
        <rect x="189" y="288" width="68" height="8"  rx="4" fill="currentColor" opacity="0.18" />
        <rect x="189" y="301" width="52" height="6"  rx="3" fill="currentColor" opacity="0.09" />

        {/* ── Feed / list section ── */}
        <rect x="76" y="324" width="52" height="9" rx="4.5" fill="currentColor" opacity="0.18" />

        {/* List item 1 */}
        <rect x="76" y="340" width="30" height="30" rx="7" fill="currentColor" opacity="0.09" />
        <rect x="114" y="344" width="92" height="8"  rx="4" fill="currentColor" opacity="0.18" />
        <rect x="114" y="358" width="140" height="6" rx="3" fill="currentColor" opacity="0.09" />
        <line x1="76" y1="376" x2="280" y2="376" stroke="currentColor" strokeWidth="0.5" opacity="0.08" />

        {/* List item 2 */}
        <rect x="76" y="382" width="30" height="30" rx="7" fill="currentColor" opacity="0.09" />
        <rect x="114" y="386" width="80" height="8"  rx="4" fill="currentColor" opacity="0.15" />
        <rect x="114" y="400" width="120" height="6" rx="3" fill="currentColor" opacity="0.08" />

        {/* ── Bottom tab bar ── */}
        <rect x="64" y="388" width="232" height="32" fill="currentColor" opacity="0.05" />
        <line x1="64" y1="388" x2="296" y2="388" stroke="currentColor" strokeWidth="0.5" opacity="0.1" />

        {/* 4 tab icons (evenly spaced across 232px) */}
        <rect x="87"  y="398" width="20" height="14" rx="4" fill="currentColor" opacity="0.14" />
        <rect x="145" y="398" width="20" height="14" rx="4" fill="currentColor" opacity="0.1"  />
        <rect x="203" y="398" width="20" height="14" rx="4" fill="currentColor" opacity="0.1"  />
        <rect x="261" y="398" width="20" height="14" rx="4" fill="currentColor" opacity="0.1"  />

        {/* Home indicator */}
        <rect x="144" y="414" width="72" height="4" rx="2" fill="currentColor" opacity="0.14" />

      </g>

      {/* ── Window frame (drawn on top of interior) ── */}
      <rect x="64" y="52" width="232" height="368" rx="16"
        stroke="currentColor" strokeWidth="1.5" fill="none" opacity="0.17" />

    </svg>
  )
}
