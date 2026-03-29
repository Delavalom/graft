"use client";

import { motion } from "framer-motion";
import {
  Wrench, Route, ArrowLeftRight, Webhook, Shield, Activity,
  Database, Users, Radio, Plug, Timer, Container, Workflow,
  GitBranch, Eye,
} from "lucide-react";
import { features } from "@/lib/constants";

const iconMap: Record<string, React.ComponentType<React.SVGProps<SVGSVGElement>>> = {
  Wrench, Route, ArrowLeftRight, Webhook, Shield, Activity,
  Database, Users, Radio, Plug, Timer, Container, Workflow,
  GitBranch, Eye,
};

export default function FeaturesGrid() {
  return (
    <section className="py-16 px-6">
      <div className="max-w-5xl mx-auto">
        <h2 className="text-2xl sm:text-3xl font-bold text-center mb-3">
          Batteries included
        </h2>
        <p className="text-muted text-sm text-center mb-10">
          15 packages. One dependency. Everything for production AI agents.
        </p>
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-2.5">
          {features.map((f, i) => {
            const Icon = iconMap[f.icon];
            return (
              <motion.div
                key={f.title}
                initial={{ opacity: 0, y: 12 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true, amount: 0.1 }}
                transition={{
                  delay: i * 0.03,
                  duration: 0.35,
                  ease: [0, 0, 0.2, 1] as [number, number, number, number],
                }}
                className="bg-dark-400/50 border border-jade-400/8 rounded-lg p-3 hover:border-jade-400/20 transition-all duration-200"
              >
                {Icon && <Icon className="w-3.5 h-3.5 text-jade-400 mb-1.5" />}
                <h3 className="text-xs font-semibold text-jade-50 mb-0.5 leading-tight">
                  {f.title}
                </h3>
                <p className="text-[11px] text-muted leading-snug">{f.description}</p>
              </motion.div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
