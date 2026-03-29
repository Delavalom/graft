import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Graft - The Go Framework for AI Agents",
  description:
    "Build production-grade AI agents in Go. Type-safe tools, multi-provider routing, OpenTelemetry observability, and durable execution. One dependency.",
  openGraph: {
    title: "Graft - The Go Framework for AI Agents",
    description:
      "Build production-grade AI agents in Go. Type-safe tools, multi-provider routing, and production observability.",
    type: "website",
    url: "https://graft.dev",
  },
  twitter: {
    card: "summary_large_image",
    title: "Graft - The Go Framework for AI Agents",
    description:
      "Build production-grade AI agents in Go. Type-safe tools, multi-provider routing, and production observability.",
  },
  keywords: [
    "go",
    "golang",
    "ai agents",
    "llm framework",
    "openai",
    "anthropic",
    "gemini",
    "opentelemetry",
    "agent orchestration",
    "type-safe",
  ],
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark" style={{ colorScheme: "dark" }}>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased bg-dark-600 text-jade-50`}
      >
        {children}
      </body>
    </html>
  );
}
