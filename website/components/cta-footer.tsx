"use client";

import { motion } from "framer-motion";
import { REPO_URL } from "@/lib/constants";

export default function CtaFooter() {
  return (
    <section className="py-20 relative overflow-hidden px-6">
      <div className="jade-glow absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[400px] -z-10 pointer-events-none" />
      <motion.div
        className="flex flex-col items-center"
        initial={{ opacity: 0, scale: 0.97 }}
        whileInView={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.4 }}
        viewport={{ once: true, amount: 0.3 }}
      >
        <h2 className="text-2xl sm:text-3xl font-bold text-center mb-5">
          Build something great with Go
        </h2>
        <div className="bg-dark-400/60 border border-jade-400/12 rounded-lg px-5 py-3 font-mono text-sm text-jade-300 text-center mb-6">
          go get github.com/delavalom/graft
        </div>
        <a
          href={REPO_URL}
          target="_blank"
          rel="noopener noreferrer"
          className="bg-jade-500 text-dark-800 rounded-full px-7 py-3 text-sm font-semibold hover:bg-jade-400 transition-colors"
        >
          Read the Docs &rarr;
        </a>
      </motion.div>
    </section>
  );
}
