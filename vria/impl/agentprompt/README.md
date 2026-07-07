# VRIA Agent Prompt — agentprompt/

**prompt_version:** `vria-prompt-v1.0`  
**Source prompt:** `vria_system_prompt_v1.md`  
**Governing specs:** `gate-b-behavior/05`, `gate-c-runtime/09`, `gate-c-runtime/10 §5`, `contracts/17 §2`

---

## Files

| File | Purpose |
|---|---|
| `vria_system_prompt_v1.md` | Production system prompt for the VRIA agent (Claude platform). Load as the system turn before any user message. |
| `README.md` | This file. Versioning policy and validation requirements. |

---

## Versioning policy

- The prompt version string is `vria-prompt-v1.0`. It is embedded as a comment in the first
  block of `vria_system_prompt_v1.md` and must be emitted verbatim in the `prompt_version`
  field of every assessment the agent produces.
- When any change is made to the system prompt — including wording, examples, rules, or tool
  guidance — increment the version string (e.g. `vria-prompt-v1.1`) in both the file header
  comment and the `<response_contract>` literal inside the prompt.
- Create a new file per version (`vria_system_prompt_v1.1.md`, etc.); do not overwrite the
  prior version. Prior versions are retained for audit traceability: every production
  assessment carries a `prompt_version` stamp that must resolve to a file in this directory.

---

## How the P7.1 harness validates prompt changes

The golden eval harness lives at `impl/goldeneval/golden_test.go`. It implements all 15 tests
from `gate-b-behavior/07_VRIA_Golden_Eval_Set.md`. The policy from 07 §5 is:

> Any change to prompt, model, scoring rules, schema, tool contract, or evidence model must
> run the golden eval suite.

Concretely, this means:

1. **Any PR that modifies a file in `impl/agentprompt/`** must trigger the full golden suite
   before merge. The CI job at `impl/ci/` (or your pipeline equivalent) runs
   `go test ./goldeneval/... -v -count=1`.

2. **All 10 critical tests** (GE-002, 003, 004, 005, 006, 007, 010, 011, 013, 015) must pass
   at 100%. A single critical failure blocks the PR.

3. **Non-critical tests** (GE-001, 008, 009, 012, 014) must pass at ≥ 90%; any failure must
   be triaged and documented before merge is approved.

4. After passing the golden suite, the PR author stamps the new `prompt_version` string in
   the commit message and in the file header. The approver verifies the stamp matches the
   file before approving.

5. The `score_value_realization` tool contract (`gate-c-runtime/09 §3.7`) requires
   `prompt_version` to be recorded in every audit row alongside `scoring_rule_version` and
   `model_version`. Updating the prompt without updating the version string breaks this
   audit trail — treat that as a blocking defect.

---

## Assessment prompt_version requirement

Every `<assessment>` block the agent emits must contain:

```yaml
prompt_version: vria-prompt-v1.0
```

This field is non-optional in `contracts/17 §7` (`ValueAssessment` schema). An assessment
missing `prompt_version` will fail schema validation and cannot enter the approval workflow.
Approval tooling rejects any draft without a resolvable prompt version.
