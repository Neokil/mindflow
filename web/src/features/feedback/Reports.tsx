import { useEffect, useState } from "react";
import { api } from "../../api";
import { ErrorMessage, Modal } from "../../ui";
import ReportForm from "./ReportForm";
import ReportThread from "./ReportThread";
import type { Report } from "./types";
export function Reports({ admin = false }: { admin?: boolean }) {
  const [reports, setReports] = useState<Report[]>([]),
    [selected, setSelected] = useState<Report>(),
    [creating, setCreating] = useState(false),
    [filter, setFilter] = useState("all"),
    [error, setError] = useState("");
  useEffect(() => {
    let disposed = false;
    async function load() {
      try {
        const reports = await api<Report[]>("/feedback");
        if (!disposed) setReports(reports);
      } catch (error) {
        if (!disposed) setError((error as Error).message);
      }
    }
    void load();
    const timer = setInterval(() => void load(), 10000);
    return () => {
      disposed = true;
      clearInterval(timer);
    };
  }, []);
  const selectedID = selected?.id;
  useEffect(() => {
    if (!selectedID) return;
    let disposed = false;
    const timer = setInterval(() => {
      void api<Report>(`/feedback/${selectedID}`)
        .then((report) => {
          if (!disposed) setSelected(report);
        })
        .catch((error) => {
          if (!disposed) setError(error.message);
        });
    }, 10000);
    return () => {
      disposed = true;
      clearInterval(timer);
    };
  }, [selectedID]);
  async function open(id: string) {
    setError("");
    try {
      setSelected(await api<Report>(`/feedback/${id}`));
    } catch (error) {
      setError((error as Error).message);
    }
  }
  function changed(report: Report) {
    setSelected(report);
    setReports((current) => [
      report,
      ...current.filter((value) => value.id !== report.id),
    ]);
    setCreating(false);
  }
  if (creating)
    return <ReportForm created={changed} cancel={() => setCreating(false)} />;
  if (selected)
    return (
      <ReportThread
        key={selected.id}
        report={selected}
        admin={admin}
        changed={changed}
        back={() => setSelected(undefined)}
      />
    );
  return (
    <section className="feedback-reports">
      <div className="form-actions">
        <button className="primary" onClick={() => setCreating(true)}>
          New report
        </button>
        <label>
          Show
          <select
            value={filter}
            onChange={(event) => setFilter(event.target.value)}
          >
            <option value="all">All reports</option>
            <option value="bug">Bugs</option>
            <option value="feature">Feature requests</option>
          </select>
        </label>
      </div>
      <p className="muted">
        {admin
          ? "Reports from all users. Reports and screenshots are private to their author and administrators."
          : "Your private reports and feature requests. Open a report to add details or read replies."}
      </p>
      <ErrorMessage message={error} />
      {reports
        .filter((report) => filter === "all" || report.kind === filter)
        .map((report) => (
          <button
            className="feedback-list-item"
            key={report.id}
            onClick={() => void open(report.id)}
          >
            <span>
              <strong>{report.title}</strong>
              <small>
                {report.kind === "bug" ? "Bug" : "Feature request"}
                {admin ? ` · ${report.authorName}` : ""}
              </small>
            </span>
            <span className="badge">{report.status}</span>
          </button>
        ))}
      {reports.length === 0 && <p className="muted">No reports yet.</p>}
    </section>
  );
}
export default function Feedback({ close }: { close: () => void }) {
  return (
    <Modal title="Feedback & Bugreport" close={close} wide>
      <Reports />
    </Modal>
  );
}
