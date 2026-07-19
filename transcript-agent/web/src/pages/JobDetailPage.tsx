import { useEffect, useRef, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { useJob } from "../api/hooks";
import { ActionChip, ErrorBox, Loading, StatusBadge } from "../components/ui";
import OverviewTab from "../tabs/OverviewTab";
import ReviewTab from "../tabs/ReviewTab";
import SummaryTab from "../tabs/SummaryTab";
import QualityTab from "../tabs/QualityTab";
import ExportsTab from "../tabs/ExportsTab";
import AuditTab from "../tabs/AuditTab";

const TABS = ["Overview", "Review", "Summary", "Quality", "Exports", "Audit"] as const;
type Tab = (typeof TABS)[number];

export default function JobDetailPage() {
  const { jobId = "" } = useParams();
  const { data: job, isLoading, error } = useJob(jobId);
  // Deep links from library search (/jobs/{id}?t=<ms>) land on the Review tab
  // so the target segment is visible immediately.
  const [searchParams] = useSearchParams();
  const [tab, setTab] = useState<Tab>(() =>
    searchParams.get("t") !== null ? "Review" : "Overview",
  );
  const qc = useQueryClient();

  // H2: when the polled job advances (status or updated_at changes), refresh
  // every derived query so a reviewer parked on any tab sees new data appear
  // without switching tabs.
  const status = job?.status;
  const updatedAt = job?.updated_at;
  const prevSignature = useRef<string | null>(null);
  useEffect(() => {
    if (!status || !updatedAt) return;
    const signature = `${status}|${updatedAt}`;
    if (prevSignature.current !== null && prevSignature.current !== signature) {
      void qc.invalidateQueries({ queryKey: ["versions", jobId] });
      void qc.invalidateQueries({ queryKey: ["segments"] });
      void qc.invalidateQueries({ queryKey: ["quality-report", jobId] });
      void qc.invalidateQueries({ queryKey: ["summary", jobId] });
      void qc.invalidateQueries({ queryKey: ["exports", jobId] });
      void qc.invalidateQueries({ queryKey: ["audit", jobId] });
      void qc.invalidateQueries({ queryKey: ["approvals", jobId] });
    }
    prevSignature.current = signature;
  }, [status, updatedAt, jobId, qc]);

  if (isLoading) return <Loading label="Loading job…" />;
  if (error) return <ErrorBox error={error} prefix="Could not load job:" />;
  if (!job) return <ErrorBox error={new Error("Job not found")} />;

  return (
    <div className="page">
      <div className="breadcrumb muted">
        <Link to="/">Jobs</Link> / <span className="mono">{job.job_id}</span>
      </div>
      <div className="page-head">
        <h1 className="job-title">
          {job.source_type === "youtube" ? "YouTube" : "Upload"} job{" "}
          <span className="mono muted">{job.job_id.slice(0, 8)}</span>
        </h1>
        <div>
          <StatusBadge status={job.status} /> <ActionChip action={job.action_required} />
        </div>
      </div>

      <div className="tabs" role="tablist">
        {TABS.map((t) => (
          <button
            key={t}
            role="tab"
            aria-selected={tab === t}
            className={tab === t ? "tab active" : "tab"}
            onClick={() => setTab(t)}
          >
            {t}
          </button>
        ))}
      </div>

      <div className="tab-panel">
        {tab === "Overview" && <OverviewTab job={job} />}
        {tab === "Review" && <ReviewTab job={job} />}
        {tab === "Summary" && <SummaryTab job={job} />}
        {tab === "Quality" && <QualityTab job={job} />}
        {tab === "Exports" && <ExportsTab job={job} />}
        {tab === "Audit" && <AuditTab job={job} />}
      </div>
    </div>
  );
}
