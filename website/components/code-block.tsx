"use client";

interface CodeBlockProps {
  code: string;
  label?: string;
  compact?: boolean;
}

function highlightGo(code: string): string {
  const lines = code.split("\n");
  return lines
    .map((line) => {
      // Check if line is a comment first
      const commentIdx = line.indexOf("//");
      if (commentIdx === 0) {
        return `<span class="text-dim">${esc(line)}</span>`;
      }

      let before = line;
      let comment = "";
      if (commentIdx > 0) {
        before = line.slice(0, commentIdx);
        comment = `<span class="text-dim">${esc(line.slice(commentIdx))}</span>`;
      }

      let result = esc(before);

      // Strings (backtick and double-quote)
      result = result.replace(
        /`([^`]*)`/g,
        '<span class="text-jade-300">`$1`</span>'
      );
      result = result.replace(
        /"([^"]*)"/g,
        '<span class="text-jade-300">"$1"</span>'
      );

      // Keywords
      result = result.replace(
        /\b(func|type|return|if|for|range|defer|go|var|const|struct|interface|package|import|nil|true|false)\b/g,
        '<span class="text-jade-400 font-medium">$1</span>'
      );

      // Types
      result = result.replace(
        /\b(string|error|context|any|int|bool|byte)\b/g,
        '<span class="text-jade-400/80">$1</span>'
      );

      return result + comment;
    })
    .join("\n");
}

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

export default function CodeBlock({ code, label, compact }: CodeBlockProps) {
  return (
    <div>
      {label && (
        <p className="text-dim text-[11px] uppercase tracking-widest mb-2 font-medium">
          {label}
        </p>
      )}
      <pre
        className={`bg-dark-400/80 border border-jade-400/10 rounded-lg border-l-2 border-l-jade-500/30 overflow-x-auto ${compact ? "p-3" : "p-4"}`}
      >
        <code
          className={`font-mono text-jade-100/90 leading-relaxed whitespace-pre block ${compact ? "text-xs" : "text-[13px]"}`}
          dangerouslySetInnerHTML={{ __html: highlightGo(code) }}
        />
      </pre>
    </div>
  );
}
