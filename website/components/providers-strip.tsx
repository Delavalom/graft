"use client";

import { motion } from "framer-motion";
import { providers } from "@/lib/constants";

export default function ProvidersStrip() {
  return (
    <section className="py-16 border-y border-jade-400/5">
      <motion.div
        initial={{ opacity: 0 }}
        whileInView={{ opacity: 1 }}
        transition={{ duration: 0.6 }}
        viewport={{ once: true }}
        className="max-w-6xl mx-auto px-6"
      >
        <p className="text-dim text-sm uppercase tracking-widest mb-6 text-center">
          Works with
        </p>
        <div className="flex flex-wrap justify-center gap-x-6 gap-y-2 items-center">
          {providers.map((name, idx) => (
            <span key={name} className="flex items-center gap-6">
              <span className="text-muted text-sm sm:text-base">{name}</span>
              {idx < providers.length - 1 && (
                <span className="text-jade-400" aria-hidden="true">
                  &middot;
                </span>
              )}
            </span>
          ))}
        </div>
      </motion.div>
    </section>
  );
}
