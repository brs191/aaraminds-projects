from __future__ import annotations

import socket

import pytest


def test_network_access_is_blocked() -> None:
    with pytest.raises(AssertionError, match="Network access is blocked"):
        socket.create_connection(("example.com", 80))
