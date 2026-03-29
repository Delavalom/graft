"use client";

import { motion } from "framer-motion";
import { useCases } from "@/lib/constants";
import CodeBlock from "./code-block";

export default function UseCases() {
  return (
    <section className="py-16 px-6">
      <h2 className="text-2xl sm:text-3xl font-bold text-center mb-10">
        What you can build
      </h2>
      <div className="grid md:grid-cols-2 gap-4 max-w-5xl mx-auto">
        {useCases.map((uc, i) => (
          <motion.div
            key={uc.title}
            initial={{ opacity: 0, y: 16 }}
            whileInView={{ opacity: 1, y: 0 }}
            viewport={{ once: true, amount: 0.2 }}
            transition={{ duration: 0.4, delay: i * 0.08 }}
            className="bg-dark-400/60 border border-jade-400/10 rounded-xl p-5 hover:border-jade-400/20 transition-all duration-200"
          >
            <h3 className="text-base font-semibold text-jade-50 mb-1">
              {uc.title}
            </h3>
            <p className="text-muted text-sm mb-3">{uc.description}</p>
            <CodeBlock code={uc.code} compact />
          </motion.div>
        ))}
      </div>
    </section>
  );
}
