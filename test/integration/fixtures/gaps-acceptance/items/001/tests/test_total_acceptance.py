"""Hidden acceptance tests for gaps item 001 (already merged, so step mode goes straight to doctor)."""
from gaps import total


def test_adds_numbers():
    assert total([1, 2, 3]) == 6
