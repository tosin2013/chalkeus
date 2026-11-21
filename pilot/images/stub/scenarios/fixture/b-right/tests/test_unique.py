from slugkit import unique_slug


def test_unique_slug_basic():
    assert unique_slug("Hello", set()) == "hello"
