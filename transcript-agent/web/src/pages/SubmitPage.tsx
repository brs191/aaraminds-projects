import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { useSubmitJob, useUploadMedia } from "../api/hooks";
import type { SourceType } from "../api/types";
import { ErrorBox } from "../components/ui";

export default function SubmitPage() {
  const navigate = useNavigate();
  const submitJob = useSubmitJob();
  const uploadMedia = useUploadMedia();
  const [sourceType, setSourceType] = useState<SourceType>("youtube");
  const [sourceUri, setSourceUri] = useState("");
  const [uploadedName, setUploadedName] = useState("");
  const [attested, setAttested] = useState(false);

  const canSubmit =
    attested && sourceUri.trim().length > 0 && !submitJob.isPending && !uploadMedia.isPending;

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    submitJob.mutate(
      {
        source_type: sourceType,
        source_uri: sourceUri.trim(),
        language: "en",
        ownership_attested: attested,
      },
      {
        onSuccess: (job) => navigate(`/jobs/${job.job_id}`),
      },
    );
  };

  return (
    <div className="page narrow">
      <h1>Submit an episode</h1>
      <form onSubmit={onSubmit} className="card form">
        <fieldset className="field">
          <legend>Source type</legend>
          <div className="toggle-group" role="radiogroup" aria-label="Source type">
            <button
              type="button"
              className={sourceType === "youtube" ? "toggle active" : "toggle"}
              onClick={() => {
                setSourceType("youtube");
                setSourceUri("");
                setUploadedName("");
              }}
            >
              YouTube URL
            </button>
            <button
              type="button"
              className={sourceType === "upload" ? "toggle active" : "toggle"}
              onClick={() => {
                setSourceType("upload");
                setSourceUri("");
                setUploadedName("");
              }}
            >
              Upload file
            </button>
          </div>
        </fieldset>

        {sourceType === "youtube" ? (
          <div className="field">
            <label htmlFor="source-uri">YouTube URL</label>
            <input
              id="source-uri"
              type="text"
              value={sourceUri}
              onChange={(e) => setSourceUri(e.target.value)}
              placeholder="https://www.youtube.com/watch?v=…"
            />
          </div>
        ) : (
          <div className="field">
            <label htmlFor="source-file">Media file</label>
            <input
              id="source-file"
              type="file"
              accept=".mp3,.m4a,.wav,.mp4,.mov,audio/*,video/mp4,video/quicktime"
              onChange={(e) => {
                const file = e.target.files?.[0];
                setSourceUri("");
                setUploadedName("");
                if (!file) return;
                uploadMedia.mutate(file, {
                  onSuccess: (res) => {
                    setSourceUri(res.source_uri);
                    setUploadedName(res.filename);
                  },
                });
              }}
            />
            {uploadMedia.isPending && <p className="muted hint">Uploading media…</p>}
            {uploadedName && (
              <p className="muted hint">
                Uploaded <span className="mono">{uploadedName}</span>.
              </p>
            )}
            <p className="muted hint">Supported formats: mp3, m4a, wav, mp4, mov.</p>
          </div>
        )}

        <div className="field">
          <label>Language</label>
          <input type="text" value="en (English only in MVP)" disabled />
        </div>

        <div className="field checkbox-field">
          <label>
            <input
              type="checkbox"
              checked={attested}
              onChange={(e) => setAttested(e.target.checked)}
            />{" "}
            I attest that our team owns or is licensed to process this content.
          </label>
          {!attested && (
            <p className="muted hint">Ownership attestation is required — the job will not be created without it.</p>
          )}
        </div>

        <ErrorBox error={uploadMedia.error} prefix="Upload failed:" />
        <ErrorBox error={submitJob.error} prefix="Submission failed:" />

        <button type="submit" className="primary" disabled={!canSubmit}>
          {submitJob.isPending ? "Submitting…" : uploadMedia.isPending ? "Uploading…" : "Submit job"}
        </button>
      </form>
    </div>
  );
}
