"""Azure AI Search (RAG) client for grounded recommendations.

Authentication:
  - AZURE_SEARCH_KEY set  → AzureKeyCredential (local dev / explicit key).
  - AZURE_SEARCH_KEY empty → DefaultAzureCredential (prefer MI in Container Apps).

On any search failure: returns (None, None, False) — never raises.
"""

from __future__ import annotations

from typing import Optional, Tuple

import structlog

from explainer.models import FindingInput, RAGSettings

log = structlog.get_logger(__name__)

# Recommendation string template
_REC_TEMPLATE = (
    "Per AT&T Network Standard §{clause}: {recommendation_text}. Source: {document_title}."
)


class RAGClient:
    """Async wrapper around azure-search-documents for finding recommendations."""

    def __init__(self, settings: RAGSettings) -> None:
        self._settings = settings
        self._client = self._build_client()

    def _build_client(self):
        """Build SearchClient with MI-preferred auth."""
        from azure.search.documents.aio import SearchClient  # type: ignore[import]
        from azure.core.credentials import AzureKeyCredential  # type: ignore[import]
        from azure.identity.aio import DefaultAzureCredential  # type: ignore[import]

        s = self._settings
        if s.azure_search_key:
            credential = AzureKeyCredential(s.azure_search_key)
            log.info("rag.auth.key_credential")
        else:
            credential = DefaultAzureCredential()
            log.info("rag.auth.managed_identity")

        return SearchClient(
            endpoint=s.azure_search_endpoint,
            index_name=s.azure_search_index,
            credential=credential,
        )

    async def search(
        self, finding: FindingInput
    ) -> Tuple[Optional[str], Optional[str], bool]:
        """Return (recommendation_text, document_title, rag_grounded).

        Queries the index for documents matching finding_type + severity.
        Returns (None, None, False) on empty results or any error.
        """
        try:
            return await self._do_search(finding)
        except Exception as exc:
            log.warning(
                "rag.search.failed",
                finding_type=finding.type,
                severity=finding.severity,
                error=str(exc),
            )
            return None, None, False

    async def _do_search(
        self, finding: FindingInput
    ) -> Tuple[Optional[str], Optional[str], bool]:
        query = f"{finding.type} {finding.severity}"
        results = []
        async with self._client as client:
            async for doc in await client.search(
                search_text=query,
                filter=None,
                top=3,
                select=["clause", "recommendation_text", "document_title", "finding_type", "severity"],
            ):
                results.append(doc)

        if not results:
            return None, None, False

        best = results[0]
        recommendation = _REC_TEMPLATE.format(
            clause=best.get("clause", "N/A"),
            recommendation_text=best.get("recommendation_text", ""),
            document_title=best.get("document_title", ""),
        )
        return recommendation, best.get("document_title"), True

    async def aclose(self) -> None:
        try:
            await self._client.close()
        except Exception:
            pass
