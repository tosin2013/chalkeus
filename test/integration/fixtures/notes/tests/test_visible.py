"""Visible tests: the package imports. Hidden acceptance tests cover each item."""


def test_package_imports():
    import notes
    assert notes.__doc__
