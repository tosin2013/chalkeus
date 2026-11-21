"""Visible tests: the package imports. Hidden acceptance tests cover each item."""


def test_package_imports():
    import tally
    assert tally.__doc__
