"use client";

import { useState, useEffect } from "react";
import { Menu, X } from "lucide-react";
import { REPO_URL } from "@/lib/constants";

const navLinks = [
  { label: "Use Cases", href: "#use-cases" },
  { label: "Features", href: "#features" },
  { label: "Docs", href: "#docs" },
  { label: "GitHub", href: REPO_URL },
];

export default function Navbar() {
  const [scrolled, setScrolled] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 20);
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  return (
    <nav
      className={`fixed top-0 left-0 right-0 z-50 backdrop-blur-xl border-b border-jade-400/10 transition-colors duration-300 ${
        scrolled ? "bg-dark-600/80" : "bg-transparent"
      }`}
    >
      <div className="max-w-6xl mx-auto px-6 flex items-center justify-between h-16">
        {/* Wordmark */}
        <a href="/" className="text-jade-400 font-bold text-xl tracking-tight">
          graft
        </a>

        {/* Desktop links */}
        <div className="hidden md:flex items-center gap-8">
          {navLinks.map((link) => (
            <a
              key={link.label}
              href={link.href}
              className="text-muted hover:text-jade-300 text-sm transition-colors"
              {...(link.href.startsWith("http")
                ? { target: "_blank", rel: "noopener noreferrer" }
                : {})}
            >
              {link.label}
            </a>
          ))}
          <a
            href="#get-started"
            className="bg-jade-500 text-dark-800 font-semibold px-5 py-2 rounded-full hover:bg-jade-400 transition-colors text-sm"
          >
            Get Started
          </a>
        </div>

        {/* Mobile hamburger */}
        <button
          className="md:hidden text-muted hover:text-jade-300 transition-colors"
          onClick={() => setMobileOpen((v) => !v)}
          aria-label="Toggle menu"
        >
          {mobileOpen ? <X size={24} /> : <Menu size={24} />}
        </button>
      </div>

      {/* Mobile dropdown */}
      {mobileOpen && (
        <div className="md:hidden border-t border-jade-400/10 bg-dark-600/95 backdrop-blur-xl">
          <div className="max-w-6xl mx-auto px-6 py-4 flex flex-col gap-4">
            {navLinks.map((link) => (
              <a
                key={link.label}
                href={link.href}
                className="text-muted hover:text-jade-300 text-sm transition-colors"
                onClick={() => setMobileOpen(false)}
                {...(link.href.startsWith("http")
                  ? { target: "_blank", rel: "noopener noreferrer" }
                  : {})}
              >
                {link.label}
              </a>
            ))}
            <a
              href="#get-started"
              className="bg-jade-500 text-dark-800 font-semibold px-5 py-2 rounded-full hover:bg-jade-400 transition-colors text-sm text-center"
              onClick={() => setMobileOpen(false)}
            >
              Get Started
            </a>
          </div>
        </div>
      )}
    </nav>
  );
}
