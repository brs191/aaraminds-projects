import { useState } from "react";
import { Link, useParams } from "react-router-dom";
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
  const [tab, setTab] = useState<Tab>("Overview");

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
