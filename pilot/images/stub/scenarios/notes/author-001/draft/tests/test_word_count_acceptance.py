"""Hidden acceptance tests for notes item 001 (canned by the integration tests' stub author)."""
from notes import word_count


def test_counts_words():
    assert word_count("a b  c") == 3


def test_empty_text():
    assert word_count("") == 0
