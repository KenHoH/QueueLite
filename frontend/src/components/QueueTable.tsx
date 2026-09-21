import { useState } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import type { QueueEntry } from "../types";
import { PriorityBadge } from "./ui";
export default function QueueTable({ entries }: { entries: QueueEntry[] }) {
  const [rows, setRows] = useState(10);
  const [page, setPage] = useState(0);
  const last = Math.max(0, Math.ceil(entries.length / rows) - 1);
  const currentPage = Math.min(page, last);
  return (
    <div className="card table-card">
      <div className="table-scroll">
        <table>
          <thead>
            <tr>
              <th>Queue number</th>
              <th>Customer</th>
              <th>Type</th>
              <th>Status</th>
              <th>Waiting time</th>
              <th>Counter</th>
            </tr>
          </thead>
          <tbody>
            {entries
              .slice(currentPage * rows, (currentPage + 1) * rows)
              .map((e) => (
                <tr key={e.id}>
                  <td className="table-ticket">{e.id}</td>
                  <td>{e.name}</td>
                  <td>
                    {e.priority ? (
                      <PriorityBadge />
                    ) : (
                      <span className="muted">Standard</span>
                    )}
                  </td>
                  <td>
                    <span
                      className={`badge ${e.status === "Serving" ? "badge-green" : "badge-gray"}`}
                    >
                      {e.status}
                    </span>
                  </td>
                  <td className="muted">
                    {e.status === "Serving"
                      ? "—"
                      : `${String(e.minutes).padStart(2, "0")} min`}
                  </td>
                  <td className="muted">
                    {e.counter ? `Counter ${e.counter}` : "Unassigned"}
                  </td>
                </tr>
              ))}
          </tbody>
        </table>
        {entries.length === 0 && (
          <div className="empty-state">
            <h3>All clear for now</h3>
            <p className="muted">No customers in this service queue.</p>
          </div>
        )}
      </div>
      <div className="table-footer">
        <span>
          {entries.length
            ? `${currentPage * rows + 1}–${Math.min((currentPage + 1) * rows, entries.length)} of ${entries.length} customers`
            : "0 customers"}
        </span>
        <div className="flex items-center gap-3">
          <label htmlFor="rows">Rows per page</label>
          <select
            id="rows"
            value={rows}
            onChange={(e) => {
              setRows(Number(e.target.value));
              setPage(0);
            }}
          >
            {[10, 25, 50].map((n) => (
              <option key={n}>{n}</option>
            ))}
          </select>
          <button
            aria-label="Previous page"
            disabled={currentPage === 0}
            onClick={() => setPage(currentPage - 1)}
          >
            <ChevronLeft size={16} />
          </button>
          <button
            aria-label="Next page"
            disabled={currentPage === last}
            onClick={() => setPage(currentPage + 1)}
          >
            <ChevronRight size={16} />
          </button>
        </div>
      </div>
    </div>
  );
}
