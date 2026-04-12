import { DocsLayout } from "fumadocs-ui/layouts/docs";
import type { ReactNode } from "react";
import { source } from "@/lib/source";
import { BookOpen, Cpu, Layers, Shield, Zap, GitBranch } from "lucide-react";

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <DocsLayout
      tree={source.pageTree}
      sidebar={{
        defaultOpenLevel: 1,
      }}
      nav={{
        title: (
          <span className="font-bold text-lg tracking-tight">
            <span className="text-fd-primary">Graft</span>{" "}
            <span className="text-fd-muted-foreground text-sm font-normal">
              Docs
            </span>
          </span>
        ),
        url: "/",
      }}
    >
      {children}
    </DocsLayout>
  );
}
