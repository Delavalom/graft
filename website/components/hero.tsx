"use client";

import { motion } from "framer-motion";
import { REPO_URL } from "@/lib/constants";

const fadeUp = {
  hidden: { opacity: 0, y: 24 },
  visible: (i: number) => ({
    opacity: 1,
    y: 0,
    transition: {
      delay: i * 0.1,
      duration: 0.5,
      ease: [0, 0, 0.2, 1] as [number, number, number, number],
    },
  }),
};

export default function Hero() {
  return (
    <section className="relative flex flex-col items-center text-center pt-28 pb-16 sm:pt-36 sm:pb-20 px-6">
      <div className="jade-glow absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[700px] h-[500px] -z-10 pointer-events-none" />

      <motion.h1
        custom={0}
        variants={fadeUp}
        initial="hidden"
        animate="visible"
        className="text-4xl sm:text-5xl lg:text-6xl font-extrabold tracking-tight leading-[1.1]"
      >
        The Go framework for{" "}
        <span className="text-jade-400 text-jade-glow">AI agents</span>
      </motion.h1>

      <motion.p
        custom={1}
        variants={fadeUp}
        initial="hidden"
        animate="visible"
        className="mt-5 text-muted text-base sm:text-lg max-w-xl"
      >
        Type-safe tools. Multi-provider routing. Durable execution. Production
        observability. One dependency.
      </motion.p>

      <motion.div
        custom={2}
        variants={fadeUp}
        initial="hidden"
        animate="visible"
        className="mt-8 flex gap-3 flex-wrap justify-center"
      >
        <a
          href="#get-started"
          className="bg-jade-500 text-dark-800 rounded-full px-7 py-3 text-sm font-semibold hover:bg-jade-400 transition-colors"
        >
          Get Started
        </a>
        <a
          href={REPO_URL}
          target="_blank"
          rel="noopener noreferrer"
          className="border border-jade-400/30 text-jade-300 rounded-full px-7 py-3 text-sm font-semibold hover:border-jade-400/60 transition-colors"
        >
          View on GitHub
        </a>
      </motion.div>
    </section>
  );
}
