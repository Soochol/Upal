import { Loader2, Check, AlertCircle, Clock } from "lucide-react";
import { Link, useLocation } from "react-router-dom";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { ThemeToggle } from "@/components/ThemeToggle";
import type { SaveStatus } from "@/hooks/useAutoSave";

type HeaderProps = {
  workflowName?: string;
  onWorkflowNameChange?: (name: string) => void;
  saveStatus?: SaveStatus;
};

const navLinks = [
  { to: "/workflows", label: "Workflows" },
  { to: "/editor", label: "Editor" },
  { to: "/runs", label: "Runs" },
  { to: "/schedules", label: "Schedules" },
  { to: "/pipelines", label: "Pipelines" },
];

export function Header({
  workflowName,
  onWorkflowNameChange,
  saveStatus,
}: HeaderProps) {
  const { pathname } = useLocation();

  return (
    <header className="flex items-center justify-between h-14 px-4 border-b border-border bg-background">
      <div className="flex items-center gap-3">
        <Link to="/" className="flex items-center gap-2 hover:opacity-80 transition-opacity">
          <div className="h-7 w-7 rounded-lg bg-primary flex items-center justify-center">
            <span className="text-primary-foreground text-sm font-bold">U</span>
          </div>
        </Link>
        <Separator orientation="vertical" className="h-6" />
        <nav className="flex items-center gap-1">
          {navLinks.map(({ to, label }) => {
            const isActive = pathname.startsWith(to);
            return (
              <Link
                key={to}
                to={to}
                className={`px-2.5 py-1.5 rounded-md text-xs font-medium transition-colors ${
                  isActive
                    ? "bg-accent text-foreground"
                    : "text-muted-foreground hover:text-foreground hover:bg-accent/50"
                }`}
              >
                {label}
              </Link>
            );
          })}
        </nav>
        {workflowName !== undefined && onWorkflowNameChange && (
          <>
            <Separator orientation="vertical" className="h-6" />
            <Input
              className="h-8 w-48 text-sm"
              placeholder="Workflow name..."
              value={workflowName}
              onChange={(e) => onWorkflowNameChange(e.target.value)}
            />
          </>
        )}
      </div>
      <div className="flex items-center gap-3">
        {saveStatus === 'waiting' && (
          <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
            <Clock className="h-3 w-3" />
            Waiting to save
          </span>
        )}
        {saveStatus === 'saving' && (
          <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
            <Loader2 className="h-3 w-3 animate-spin" />
            Saving...
          </span>
        )}
        {saveStatus === 'saved' && (
          <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
            <Check className="h-3 w-3" />
            Saved
          </span>
        )}
        {saveStatus === 'error' && (
          <span className="flex items-center gap-1 text-[10px] text-destructive">
            <AlertCircle className="h-3 w-3" />
            Save failed
          </span>
        )}
        <ThemeToggle />
      </div>
    </header>
  );
}
