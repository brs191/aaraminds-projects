import { useRef, useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { ApiError } from "../api/client";
import { useSubmitJob, useUploadMedia } from "../api/hooks";
import type { SourceType } from "../api/types";
import { ErrorBox } from "../components/ui";

const ACCEPTED_EXTENSIONS = [".mp3", ".m4a", ".wav", ".mp4", ".mov"];

function uploadErrorMessage(error: unknown): string | null {
  if (!(error instanceof ApiError)) return null;
  if (error.code === "UNSUPPORTED_FORMAT" || error.status === 400) {
    return `That file type is not supported. Allowed formats: ${ACCEPTED_EXTENSIONS.join(", ")}.`;
  }
  if (error.code === "REQUEST_TOO_LARGE" || error.status === 413) {
    return "That file is too large to upload. Pick a smaller file or trim the media first.";
  }
  return null;
}

export default function SubmitPage() {
  const navigate = useNavigate();
  const submitJob = useSubmitJob();
  const uploadMedia = useUploadMedia();
  const [sourceType, setSourceType] = useState<SourceType>("youtube");
  const [sourceUri, setSourceUri] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [advancedUri, setAdvancedUri] = useState(false);
  const [attested, setAttested] = useState(false);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const usingFileUpload = sourceType === "upload" && !advancedUri;
  const busy = submitJob.isPending || uploadMedia.isPending;
  const hasSource = usingFileUpload ? file !== null : sourceUri.trim().length > 0;
  const canSubmit = attested && hasSource && !busy;

  const resetSource = () => {
    setSourceUri("");
    setFile(null);
    uploadMedia.reset();
    submitJob.reset();
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  const createJob = (uri: string) => {
    submitJob.mutate(
      {
        source_type: sourceType,
        source_uri: uri,
        language: "en",
        ownership_attested: attested,
      },
      {
        onSuccess: (job) => navigate(`/jobs/${job.job_id}`),
      },
    );
  };

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    if (usingFileUpload && file) {
      // Upload first (multipart), then create the job with the returned upload_uri.
      uploadMedia.mutate(file, {
        onSuccess: (res) => createJob(res.upload_uri),
      });
      return;
    }
    createJob(sourceUri.trim());
  };

  const friendlyUploadError = uploadErrorMessage(uploadMedia.error);

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
                setAdvancedUri(false);
                resetSource();
              }}
            >
              YouTube URL
            </button>
            <button
              type="button"
              className={sourceType === "upload" ? "toggle active" : "toggle"}
              onClick={() => {
                setSourceType("upload");
                resetSource();
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
        ) : usingFileUpload ? (
          <div className="field">
            <label htmlFor="source-file">Media file</label>
            <input
              id="source-file"
              type="file"
              ref={fileInputRef}
              accept={ACCEPTED_EXTENSIONS.join(",")}
              onChange={(e) => {
                uploadMedia.reset();
                setFile(e.target.files?.[0] ?? null);
              }}
            />
            {file && (
              <p className="muted hint">
                Selected <span className="mono">{file.name}</span> (
                {(file.size / (1024 * 1024)).toFixed(1)} MB). Uploads when you submit.
              </p>
            )}
            <p className="muted hint">Supported formats: mp3, m4a, wav, mp4, mov.</p>
            <button
              type="button"
              className="link-button"
              onClick={() => {
                setAdvancedUri(true);
                resetSource();
              }}
            >
              Advanced: paste an upload URI instead
            </button>
          </div>
        ) : (
          <div className="field">
            <label htmlFor="source-uri">Upload URI (advanced)</label>
            <input
              id="source-uri"
              type="text"
              value={sourceUri}
              onChange={(e) => setSourceUri(e.target.value)}
              placeholder="upload://… or mock://…"
            />
            <p className="muted hint">
              For mock/demo URIs or an <span className="mono">upload://</span> URI from a previous
              upload. No file is transferred.
            </p>
            <button
              type="button"
              className="link-button"
              onClick={() => {
                setAdvancedUri(false);
                resetSource();
              }}
            >
              Back to file upload
            </button>
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

        {uploadMedia.isPending && (
          <div className="notice-banner upload-progress" role="status">
            Uploading <span className="mono">{file?.name}</span>… this can take a while for large
            files.
          </div>
        )}

        {friendlyUploadError ? (
          <div className="error-box" role="alert">
            <strong>Upload failed:</strong> {friendlyUploadError}
          </div>
        ) : (
          <ErrorBox error={uploadMedia.error} prefix="Upload failed:" />
        )}
        <ErrorBox error={submitJob.error} prefix="Submission failed:" />

        <button type="submit" className="primary" disabled={!canSubmit}>
          {uploadMedia.isPending
            ? "Uploading…"
            : submitJob.isPending
              ? "Submitting…"
              : "Submit job"}
        </button>
      </form>
    </div>
  );
}
