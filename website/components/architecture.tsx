"use client";

import { motion } from "framer-motion";
import { ArrowRight } from "lucide-react";

const steps = ["Messages", "Agent", "Runner", "LLM", "Tools", "Result"];

export default function Architecture() {
  return (
    <section className="py-14 bg-dark-500 px-6">
      <h2 className="text-2xl sm:text-3xl font-bold text-center mb-3">
        How it works
      </h2>
      <motion.div
        initial={{ opacity: 0 }}
        whileInView={{ opacity: 1 }}
        transition={{ duration: 0.4 }}
        viewport={{ once: true }}
      >
        <div className="flex items-center justify-center flex-wrap gap-1 mt-8">
          {steps.map((step, i) => (
            <div key={step} className="flex items-center">
              <span className="inline-flex items-center px-3 py-1.5 border border-jade-400/15 rounded-md text-xs font-medium text-jade-100 bg-dark-400/60">
                {step}
              </span>
              {i < steps.length - 1 && (
                <ArrowRight className="text-jade-500/60 mx-1 shrink-0" size={14} />
              )}
            </div>
          ))}
        </div>
        <p className="text-muted text-xs text-center max-w-lg mx-auto mt-4">
          Generate, execute tool calls, repeat. Handoffs switch agents mid-loop.
        </p>
      </motion.div>
    </section>
  );
}
