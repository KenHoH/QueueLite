import { createContext, useContext, useState } from "react";
import type { ReactNode } from "react";
import type { QueueEntry, QueueStatus } from "../types";
import { initialEntries } from "../data/mock";
type Outcome = Extract<QueueStatus, "Completed" | "Skipped" | "Cancelled">;
interface QueueState {
  entries: QueueEntry[];
  join: (businessId: string, name: string) => string;
  advance: (counter: number, outcome?: Outcome) => void;
  totals: { Completed: number; Skipped: number; Cancelled: number };
}
const Context = createContext<QueueState | null>(null);
export function QueueProvider({ children }: { children: ReactNode }) {
  const [entries, setEntries] = useState(initialEntries);
  function join(businessId: string, name: string) {
    const existing = entries.find(
      (e) =>
        e.businessId === businessId &&
        e.mine &&
        ["Waiting", "Serving"].includes(e.status),
    );
    if (existing) return existing.id;
    const prefix =
      businessId === "northside" ? "A" : businessId === "greencare" ? "G" : "Q";
    const next =
      Math.max(
        0,
        ...entries
          .filter((e) => e.id.startsWith(`${prefix}-`))
          .map((e) => Number(e.id.split("-")[1])),
      ) + 1;
    const id = `${prefix}-${String(next).padStart(3, "0")}`;
    setEntries((prev) => [
      ...prev,
      {
        id,
        businessId,
        name,
        priority: false,
        status: "Waiting",
        service: "General",
        minutes: 0,
        mine: true,
      },
    ]);
    return id;
  }
  function advance(counter: number, outcome?: Outcome) {
    const startedAt = new Date().toLocaleTimeString("en-GB", {
      hour: "2-digit",
      minute: "2-digit",
    });
    setEntries((prev) => {
      const current = prev.find(
        (e) =>
          e.businessId === "northside" &&
          e.counter === counter &&
          e.status === "Serving",
      );
      if (current && !outcome) return prev;
      const next = prev.find(
        (e) => e.businessId === "northside" && e.status === "Waiting",
      );
      return prev.map((e) =>
        e.id === current?.id
          ? { ...e, status: outcome!, counter: undefined }
          : e.id === next?.id
            ? { ...e, status: "Serving", counter, startedAt }
            : e,
      );
    });
  }
  const totals = { Completed: 30, Skipped: 2, Cancelled: 1 };
  entries.forEach((e) => {
    if (e.businessId === "northside" && e.status in totals)
      totals[e.status as Outcome]++;
  });
  return (
    <Context.Provider value={{ entries, join, advance, totals }}>
      {children}
    </Context.Provider>
  );
}
export function useQueue() {
  const value = useContext(Context);
  if (!value) throw new Error("QueueProvider is required");
  return value;
}
