"use client";

import { motion } from "framer-motion";
import { dxSnippets } from "@/lib/constants";
import CodeBlock from "./code-block";

export default function DxShowcase() {
  return (
    <section className="py-16 bg-dark-500 px-6">
      <h2 className="text-2xl sm:text-3xl font-bold text-center mb-2">
        Developer experience, not boilerplate
      </h2>
      <p className="text-muted text-sm text-center mb-10">
        Minimal code. Maximum capability.
      </p>
      <div className="grid md:grid-cols-3 gap-4 max-w-5xl mx-auto">
        {dxSnippets.map((snippet, i) => (
          <motion.div
            key={snippet.label}
            initial={{ opacity: 0, y: 16 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true, amount: 0.2 }}
            transition={{ duration: 0.4, delay: i * 0.08 }}
          >
            <CodeBlock label={snippet.label} code={snippet.code} compact />
          </motion.div>
        ))}
      </div>
    </section>
  );
}
