// src/app/layout.tsx
import { useState, useEffect } from 'react';
import type { ReactNode } from 'react';
import { Zap, Box, Activity, Settings, Workflow, Globe, Menu, Inbox, Send, X } from 'lucide-react';
import { useLocation } from 'react-router-dom';
import { useResizeDrag } from '@/shared/lib/useResizeDrag';
import { useContentSessionStore } from '@/entities/content-session/store';
import { NavLink } from 'react-router-dom';
import { cn } from '@/shared/lib/utils';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/shared/ui/tooltip';
import { Separator } from '@/shared/ui/separator';

interface MainLayoutProps {
    children: ReactNode;
    headerContent?: ReactNode;
    rightPanel?: ReactNode;
    bottomConsole?: ReactNode;
}

const NAV_ITEMS = [
    { icon: Inbox, label: 'Review Inbox', to: '/inbox' },
    { icon: Send, label: 'Publish Inbox', to: '/publish-inbox' },
    { icon: Box, label: 'Workflows', to: '/workflows' },
    { icon: Workflow, label: 'Pipelines', to: '/pipelines' },
    { icon: Globe, label: 'Published', to: '/published' },
    { icon: Activity, label: 'Runs', to: '/runs' },
    { icon: Zap, label: 'Connections', to: '/connections' },
];

export function MainLayout({ children, headerContent, rightPanel, bottomConsole }: MainLayoutProps) {
    const sidebarExpanded = false;
    const [gnbVisible, setGnbVisible] = useState(true);
    const [mobileNavOpen, setMobileNavOpen] = useState(false);
    const location = useLocation();
    const pendingCount = useContentSessionStore((s) => s.pendingCount);
    const publishReadyCount = useContentSessionStore((s) => s.publishReadyCount);
    const syncBadgeCounts = useContentSessionStore((s) => s.syncBadgeCounts);

    const badgeMap: Record<string, number> = {
        '/inbox': pendingCount,
        '/publish-inbox': publishReadyCount,
    };

    useEffect(() => {
        syncBadgeCounts();
    }, [syncBadgeCounts]);

    const { size: rightPanelWidth, handleMouseDown: onRightPanelDrag } = useResizeDrag({
        direction: 'horizontal',
        min: 260,
        max: 700,
        initial: 320,
    });

    // Close mobile nav on route change
    useEffect(() => {
        setMobileNavOpen(false);
    }, [location.pathname]);

    return (
        <div className="flex h-screen w-screen overflow-hidden bg-background text-foreground selection:bg-primary/20">

            {/* 1. Global Navigation Bar (GNB) — hidden on mobile, sidebar on desktop */}
            <nav className={cn(
                "hidden md:flex flex-col py-4 bg-sidebar border-r border-sidebar-border z-50 transition-all duration-300 ease-[cubic-bezier(0.2,0.8,0.2,1)] overflow-hidden shadow-xl shrink-0",
                gnbVisible ? "w-[240px]" : "w-[64px] items-center"
            )}>
                <div className="mb-8 px-4 flex flex-row items-center gap-3 font-bold text-xl text-primary font-heading overflow-hidden whitespace-nowrap">
                    {/* Logo Placeholder */}
                    <div className="size-8 min-w-8 rounded-lg bg-primary text-primary-foreground flex items-center justify-center">U</div>
                    {gnbVisible && (
                        <span className="transition-all duration-300 animate-in fade-in slide-in-from-left-2 overflow-hidden">
                            Upal
                        </span>
                    )}
                </div>

                <TooltipProvider delayDuration={0}>
                    <div className="flex flex-col gap-2 flex-1 px-2">
                        {NAV_ITEMS.map((item) => (
                            <Tooltip key={item.to}>
                                <TooltipTrigger asChild>
                                    <NavLink
                                        to={item.to}
                                        className={({ isActive }) =>
                                            cn(
                                                "relative block px-3 py-2.5 rounded-xl transition-all duration-200 group overflow-hidden",
                                                isActive
                                                    ? "bg-primary/10 text-primary shadow-sm"
                                                    : "text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
                                            )
                                        }
                                    >
                                        <div className="flex flex-row items-center gap-3 whitespace-nowrap">
                                            <span className="relative shrink-0">
                                                <item.icon className="size-5 min-w-5" />
                                                {(badgeMap[item.to] ?? 0) > 0 && (
                                                    <span className="absolute -top-1.5 -right-2 flex items-center justify-center min-w-[18px] h-[18px] px-1 rounded-full bg-destructive text-destructive-foreground text-[10px] font-bold leading-none">
                                                        {badgeMap[item.to] > 99 ? '99+' : badgeMap[item.to]}
                                                    </span>
                                                )}
                                            </span>
                                            {gnbVisible && (
                                                <span className="text-sm font-medium">
                                                    {item.label}
                                                </span>
                                            )}
                                        </div>
                                    </NavLink>
                                </TooltipTrigger>
                                {!gnbVisible && <TooltipContent side="right" className="font-medium">{item.label}</TooltipContent>}
                            </Tooltip>
                        ))}
                    </div>
                </TooltipProvider>

                <div className="mt-auto px-2 flex flex-col gap-2">
                    <button className="block px-3 py-2.5 rounded-xl text-muted-foreground hover:bg-sidebar-accent transition-colors overflow-hidden">
                        <div className="flex flex-row items-center gap-3 whitespace-nowrap">
                            <Settings className="size-5 min-w-5 shrink-0" />
                            {gnbVisible && (
                                <span className="text-sm font-medium">Settings</span>
                            )}
                        </div>
                    </button>
                </div>
            </nav>

            {/* Mobile Navigation Overlay */}
            {mobileNavOpen && (
                <div className="fixed inset-0 z-[100] md:hidden">
                    <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={() => setMobileNavOpen(false)} />
                    <nav className="absolute inset-y-0 left-0 w-[280px] bg-sidebar border-r border-sidebar-border shadow-2xl flex flex-col py-4 animate-in slide-in-from-left duration-200">
                        <div className="mb-6 px-4 flex items-center justify-between">
                            <div className="flex items-center gap-3 font-bold text-xl text-primary font-heading">
                                <div className="size-8 min-w-8 rounded-lg bg-primary text-primary-foreground flex items-center justify-center">U</div>
                                <span>Upal</span>
                            </div>
                            <button onClick={() => setMobileNavOpen(false)} className="p-1.5 rounded-lg text-muted-foreground hover:bg-sidebar-accent transition-colors">
                                <X className="size-5" />
                            </button>
                        </div>
                        <div className="flex flex-col gap-1 flex-1 px-2">
                            {NAV_ITEMS.map((item) => (
                                <NavLink
                                    key={item.to}
                                    to={item.to}
                                    className={({ isActive }) =>
                                        cn(
                                            "block px-3 py-2.5 rounded-xl transition-all duration-200",
                                            isActive
                                                ? "bg-primary/10 text-primary shadow-sm"
                                                : "text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
                                        )
                                    }
                                >
                                    <div className="flex items-center gap-3">
                                        <span className="relative shrink-0">
                                            <item.icon className="size-5" />
                                            {(badgeMap[item.to] ?? 0) > 0 && (
                                                <span className="absolute -top-1.5 -right-2 flex items-center justify-center min-w-[18px] h-[18px] px-1 rounded-full bg-destructive text-destructive-foreground text-[10px] font-bold leading-none">
                                                    {badgeMap[item.to] > 99 ? '99+' : badgeMap[item.to]}
                                                </span>
                                            )}
                                        </span>
                                        <span className="text-sm font-medium">{item.label}</span>
                                    </div>
                                </NavLink>
                            ))}
                        </div>
                        <div className="px-2">
                            <button className="block w-full px-3 py-2.5 rounded-xl text-muted-foreground hover:bg-sidebar-accent transition-colors">
                                <div className="flex items-center gap-3">
                                    <Settings className="size-5 shrink-0" />
                                    <span className="text-sm font-medium">Settings</span>
                                </div>
                            </button>
                        </div>
                    </nav>
                </div>
            )}

            {/* Main Content Area */}
            <div className="flex flex-col flex-1 min-w-0 h-full relative z-0">

                <header className="h-[56px] min-h-[56px] border-b border-border bg-background/80 backdrop-blur-md flex items-center px-4 justify-between z-40 sticky top-0">
                    <div className="flex-1 flex items-center gap-3 font-medium">
                        {/* Desktop: toggle sidebar, Mobile: open drawer */}
                        <button
                            onClick={() => {
                                // mobile
                                if (window.innerWidth < 768) {
                                    setMobileNavOpen(true);
                                } else {
                                    setGnbVisible(!gnbVisible);
                                }
                            }}
                            className={cn(
                                "p-2 -ml-2 rounded-lg transition-all duration-200",
                                gnbVisible
                                    ? "md:bg-primary/10 md:text-primary text-muted-foreground"
                                    : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                            )}
                            title={gnbVisible ? "Hide Menu" : "Show Menu"}
                        >
                            <Menu className="size-5" />
                        </button>
                        <Separator orientation="vertical" className="h-4" />
                        {headerContent || <span className="text-muted-foreground">Upal Workspace</span>}
                    </div>
                </header>

                {/* 3. Main Body + Panels Container */}
                <div className="flex flex-1 overflow-hidden relative">

                    {/* Sub Sidebar / Palette (expandable) */}
                    <div className={cn(
                        "h-full border-r border-border bg-sidebar/50 backdrop-blur-sm shadow-sm transition-all duration-300 ease-[cubic-bezier(0.2,0.8,0.2,1)] will-change-[width] z-20 flex flex-col",
                        sidebarExpanded ? "w-[260px] opacity-100" : "w-0 opacity-0 overflow-hidden"
                    )}>
                        <div className="p-4 border-b border-border min-w-[260px]">
                            <h3 className="font-semibold text-sm text-muted-foreground uppercase tracking-wider">Explorer</h3>
                        </div>
                        <div className="flex-1 overflow-y-auto p-2 min-w-[260px]">
                            {/* Left Context Content (Nodes palette, file tree, etc.) */}
                            <div className="text-sm text-muted-foreground p-2">Select an item...</div>
                        </div>
                    </div>

                    {/* Central Canvas / Main View */}
                    <main className="flex-1 flex flex-col relative h-full bg-grid-pattern overflow-hidden">
                        {children}
                    </main>

                    {/* 4. Right Inspector Panel — hidden on mobile */}
                    {rightPanel && (
                        <div className="hidden md:contents">
                            <div
                                onMouseDown={onRightPanelDrag}
                                className="w-1 shrink-0 cursor-col-resize hover:bg-primary/30 active:bg-primary/50 transition-colors z-30 relative
                                    after:absolute after:inset-y-0 after:-left-1 after:-right-1"
                            />
                            <aside
                                style={{ width: rightPanelWidth }}
                                className="border-l border-border bg-sidebar/95 backdrop-blur-md shadow-2xl z-30 flex flex-col shrink-0"
                            >
                                {rightPanel}
                            </aside>
                        </div>
                    )}

                </div>

                {/* 5. Bottom Console */}
                {bottomConsole}

            </div>

        </div>
    );
}
