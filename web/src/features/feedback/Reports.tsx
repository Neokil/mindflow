import { useEffect, useState } from "react";
import { api } from "../../api";
import { ErrorMessage, Modal } from "../../ui";
import ReportForm from "./ReportForm";
import ReportThread from "./ReportThread";
import { statusLabel } from "./types";
import type { Report } from "./types";
import { feedbackChanged } from "./useFeedbackUnread";
export function Reports({ admin = false }: { admin?: boolean }) {
  const [reports, setReports] = useState<Report[]>([]),
    [selected, setSelected] = useState<Report>(),
    [creating, setCreating] = useState(false),
    [error, setError] = useState(""),
    [search, setSearch] = useState("");
  const endpoint = admin
    ? `/feedback?scope=all&q=${encodeURIComponent(search)}`
    : "/feedback";
  useEffect(() => {
    let disposed = false;
    const load = () => {
      void api<Report[]>(endpoint)
        .then((values) => {
          if (!disposed) setReports(values);
        })
        .catch((error) => {
          if (!disposed) setError(error.message);
        });
    };
    load();
    const timer = setInterval(load, 10000);
    return () => {
      disposed = true;
      clearInterval(timer);
    };
  }, [endpoint]);
  const selectedID = selected?.id;
  const through = selected?.readThrough;
  useEffect(() => {
    if (!selectedID || through === undefined) return;
    let disposed = false;
    void api(`/feedback/${selectedID}/read`, "POST", { through })
      .then(() => {
        if (disposed) return;
        setReports((values) =>
          values.map((report) =>
            report.id === selectedID ? { ...report, unread: 0 } : report,
          ),
        );
        feedbackChanged();
      })
      .catch((error) => {
        if (!disposed) setError(error.message);
      });
    return () => {
      disposed = true;
    };
  }, [selectedID, through]);
  useEffect(() => {
    if (!selectedID) return;
    let disposed = false;
    const timer = setInterval(() => {
      if (document.visibilityState === "hidden") return;
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
    if (creating) setSearch("");
    setReports((values) => [
      report,
      ...values.filter((value) => value.id !== report.id),
    ]);
    setCreating(false);
    feedbackChanged();
  }
  return (
    <section className="feedback-reports">
      <div className="feedback-list-actions">
        {admin && (
          <label>
            Search tickets
            <input
              type="search"
              maxLength={200}
              value={search}
              placeholder="Title, description, or comments"
              onChange={(event) => setSearch(event.target.value)}
            />
          </label>
        )}
        <button className="primary" onClick={() => setCreating(true)}>
          New report
        </button>
      </div>
      <p className="muted">
        {admin
          ? "Reports from all users. Reports and screenshots are private to their author and administrators."
          : "Your private reports and feature requests. Open a report to add details or read replies."}
      </p>
      <ErrorMessage message={error} />
      {reports.map((report) => (
        <button
          className={`feedback-list-item${report.unread ? " has-unread" : ""}`}
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
          <span>
            {report.unread > 0 && (
              <span
                className="notification-badge"
                aria-label={`${report.unread} unread updates`}
              >
                {report.unread}
              </span>
            )}{" "}
            <span className="badge">{statusLabel(report.status)}</span>
          </span>
        </button>
      ))}
      {reports.length === 0 && (
        <p className="muted">
          {search.trim() ? "No reports match your search." : "No reports yet."}
        </p>
      )}
      {creating && (
        <Modal
          title="New report"
          close={() => setCreating(false)}
          closeLabel="Back to reports"
          wide
        >
          <ReportForm created={changed} cancel={() => setCreating(false)} />
        </Modal>
      )}
      {selected && (
        <Modal
          title={selected.title}
          close={() => setSelected(undefined)}
          closeLabel="Back to reports"
          wide
        >
          <ReportThread
            key={selected.id}
            report={selected}
            admin={admin}
            changed={changed}
          />
        </Modal>
      )}
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
