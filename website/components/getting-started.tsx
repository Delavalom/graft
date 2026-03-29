"use client";

import { motion } from "framer-motion";

const steps = [
  { label: "Install", code: `go get github.com/delavalom/graft` },
  {
    label: "Define your agent",
    code: `agent := graft.NewAgent("assistant",
  graft.WithInstructions("You are helpful."),
  graft.WithTools(myTool),
)`,
  },
  {
    label: "Run it",
    code: `runner := graft.NewDefaultRunner(model)
result, _ := runner.Run(ctx, agent, messages)
fmt.Println(result.LastAssistantText())`,
  },
];

export default function GettingStarted() {
  return (
    <section id="get-started" className="py-16 px-6">
      <h2 className="text-2xl sm:text-3xl font-bold text-center mb-10">
        Start in 30 seconds
      </h2>
      <motion.div
        className="max-w-xl mx-auto"
        initial={{ opacity: 0, y: 16 }}
        whileInView={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4 }}
        viewport={{ once: true, amount: 0.2 }}
      >
        {steps.map((step, i) => (
          <div key={step.label} className="flex gap-3 mb-5">
            <div className="w-6 h-6 rounded-full bg-jade-500/15 text-jade-400 font-bold text-[11px] flex items-center justify-center shrink-0 mt-0.5">
              {i + 1}
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-xs font-semibold text-jade-50 mb-1.5">
                {step.label}
              </p>
              <pre className="bg-dark-400/60 border border-jade-400/10 rounded-lg p-3 font-mono text-xs text-jade-100 overflow-x-auto leading-relaxed">
                <code>{step.code}</code>
              </pre>
            </div>
          </div>
        ))}
      </motion.div>
    </section>
  );
}
