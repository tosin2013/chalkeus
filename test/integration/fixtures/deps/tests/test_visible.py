"""Visible test: the code supports the greetlib major pinned in requirements.txt."""
import greet


def test_supports_the_pinned_greetlib():
    assert greet.pinned_major() in greet.SUPPORTED
