-- Generated from contracts/19_VRIA_Physical_Data_Model.md (v1.3). Do not hand-edit; regenerate.

create table assessment_evidence (
  assessment_id uuid not null references value_assessments(assessment_id),
  evidence_source_id uuid not null references evidence_sources(evidence_source_id),
  citation_pointer text,
  primary key (assessment_id, evidence_source_id)
);
