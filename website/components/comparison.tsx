"use client";

import { motion } from "framer-motion";
import { comparison } from "@/lib/constants";

export default function Comparison() {
  return (
    <section className="py-16 px-6">
      <h2 className="text-2xl sm:text-3xl font-bold text-center mb-3">
        Why Graft?
      </h2>
      <p className="text-muted text-sm text-center mb-10">
        The only comprehensive Go-native AI agent framework.
      </p>
      <motion.div
        className="max-w-4xl mx-auto overflow-x-auto"
        initial={{ opacity: 0, y: 16 }}
        whileInView={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4 }}
        viewport={{ once: true, amount: 0.2 }}
      >
        <table className="w-full border-collapse min-w-[580px] text-[13px]">
          <thead>
            <tr>
              {comparison.headers.map((header, i) => (
                <th
                  key={header || "label"}
                  className={`text-left font-semibold py-2.5 px-3 ${
                    i === 1 ? "text-jade-400 bg-jade-400/5" : i === 0 ? "" : "text-muted"
                  }`}
                >
                  {header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {comparison.rows.map((row, rowIdx) => (
              <tr
                key={row.label}
                className={`border-b border-jade-400/5 ${rowIdx % 2 === 0 ? "bg-dark-400/40" : ""}`}
              >
                <td className="text-muted font-medium py-2 px-3">{row.label}</td>
                {row.values.map((value, colIdx) => (
                  <td
                    key={colIdx}
                    className={
                      colIdx === 0
                        ? "text-jade-300 font-medium py-2 px-3 bg-jade-400/5"
                        : "text-dim py-2 px-3"
                    }
                  >
                    {colIdx === 0 && row.graftWins && (
                      <span className="w-1.5 h-1.5 rounded-full bg-jade-400 inline-block mr-1.5 align-middle" />
                    )}
                    {value}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </motion.div>
    </section>
  );
}
