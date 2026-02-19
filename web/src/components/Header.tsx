import { Play, Save, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { ThemeToggle } from "@/components/ThemeToggle";
import { cn } from "@/lib/utils";

type HeaderProps = {
  workflowName: string;
  onWorkflowNameChange: (name: string) => void;
  onSave: () => void;
  onRun: () => void;
  onGenerate: () => void;
  isRunning: boolean;
};

export function Header({
  workflowName,
  onWorkflowNameChange,
  onSave,
  onRun,
  onGenerate,
  isRunning,
}: HeaderProps) {
  return (
    <header className="flex items-center justify-between h-14 px-4 border-b border-border bg-background">
      <div className="flex items-center gap-3">
        <div className="h-7 w-7 rounded-lg bg-primary flex items-center justify-center">
          <span className="text-primary-foreground text-sm font-bold">U</span>
        </div>
        <span className="text-sm font-semibold text-foreground">Upal</span>
        <Separator orientation="vertical" className="h-6" />
        <Input
          className="h-8 w-48 text-sm"
          placeholder="Workflow name..."
          value={workflowName}
          onChange={(e) => onWorkflowNameChange(e.target.value)}
        />
      </div>
      <div className="flex items-center gap-2">
        <Button variant="outline" size="sm" onClick={onGenerate}>
          <Sparkles className="h-4 w-4 mr-1.5" />
          Generate
        </Button>
        <Button variant="outline" size="sm" onClick={onSave}>
          <Save className="h-4 w-4 mr-1.5" />
          Save
        </Button>
        <Button
          size="sm"
          className={cn(
            "bg-node-output text-node-output-foreground hover:bg-node-output/90"
          )}
          disabled={isRunning}
          onClick={onRun}
        >
          <Play className="h-4 w-4 mr-1.5" />
          {isRunning ? "Running..." : "Run"}
        </Button>
        <Separator orientation="vertical" className="h-6 mx-1" />
        <ThemeToggle />
      </div>
    </header>
  );
}
