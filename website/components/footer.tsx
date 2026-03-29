"use client";

import { REPO_URL } from "@/lib/constants";

export default function Footer() {
  return (
    <footer className="border-t border-jade-400/10">
      <div className="max-w-6xl mx-auto px-6 py-12 flex flex-col sm:flex-row justify-between items-center gap-4">
        {/* Wordmark */}
        <span className="text-jade-400 font-bold text-lg">graft</span>

        {/* Right links */}
        <div className="flex items-center gap-6 text-dim text-sm">
          <span>MIT License</span>
          <a
            href={REPO_URL}
            target="_blank"
            rel="noopener noreferrer"
            className="hover:text-jade-300 transition-colors"
          >
            GitHub
          </a>
        </div>
      </div>
    </footer>
  );
}
