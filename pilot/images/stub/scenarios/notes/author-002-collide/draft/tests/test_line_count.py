"""A draft at a path the item's own plan names: the check must refuse it (DRAFT_PATH_IN_SPEC)."""
from notes import line_count


def test_counts_lines():
    assert line_count("a\nb\n") == 2
