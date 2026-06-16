

#### From CODEX
I need you to execute this project systematically and methodically, using the roadmap as the source of truth.

Primary inputs:
- `adoption_roadmap.md`
- `adoption_roadmap.svg`

First, review both files carefully. Extract all waves, milestones, dependencies, deliverables, risks, and implied work items. Treat this as a multi-engineer execution effort with Tier 1 enterprise-grade quality.

Operating approach:
1. Discovery
   - Read and understand the existing repository/project structure.
   - Review the roadmap documents.
   - Identify assumptions, gaps, dependencies, and risks.
   - Search the internet where needed, prioritizing official documentation, standards, vendor docs, and current best practices.
   - Cite external references used.

2. Execution plan
   - Produce a clear implementation plan organized by roadmap wave.
   - Break each wave into actionable tasks.
   - Identify parallelizable work and assign it to sub-agents where useful.
   - Define acceptance criteria for each wave.
   - Do not begin large implementation until the plan is coherent.

3. Implementation
   - Execute each wave in order unless dependencies allow safe parallel work.
   - Use sub-agents for architecture review, implementation, QA, documentation, security review, and research as needed.
   - Maintain consistency with the existing codebase and architecture.
   - Avoid unnecessary rewrites or unrelated refactors.
   - Preserve existing user changes.

4. Quality assurance
   - Add or update tests appropriate to the risk and scope.
   - Run available quality gates: tests, linting, type checks, builds, security checks, and any existing validation scripts.
   - Validate edge cases, failure paths, and integration points.
   - For frontend/UI work, verify responsiveness, accessibility, visual consistency, and no layout regressions.

5. Documentation
   - Update relevant documentation, README files, architecture notes, runbooks, or decision records.
   - Document assumptions, tradeoffs, and any remaining risks.

6. Final report
   - Summarize completed work by wave.
   - List files changed.
   - List tests/checks run and their results.
   - Note unresolved issues, risks, or follow-up recommendations.
   - Include external references used.

Quality bar:
- Treat this as enterprise-grade production work.
- Be thorough, but keep changes scoped to the roadmap.
- Prefer maintainable, boring, well-tested solutions over clever ones.
- If something is ambiguous, make a reasonable assumption and document it. Ask for clarification only if progress would be risky without it.
- Use as many sub-agents and tokens as needed to achieve high quality, but keep the main agent responsible for final integration and consistency.

#### From Claude with My Agents
