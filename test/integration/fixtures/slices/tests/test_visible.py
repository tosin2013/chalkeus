"""Visible tests: the package imports. Hidden acceptance tests cover each slice."""


def test_package_imports():
    import stats
    assert stats.__doc__
