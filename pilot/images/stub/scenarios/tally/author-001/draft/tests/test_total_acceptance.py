"""Hidden acceptance tests for tally item 001 (canned by the integration tests' stub author)."""
from tally import total


def test_adds_numbers():
    assert total([1, 2, 3]) == 6


def test_empty_list():
    assert total([]) == 0
