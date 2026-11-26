import greet


def test_supports_greetlib_2():
    assert 2 in greet.SUPPORTED


def test_greets_as_greetlib_2():
    assert greet.greet("Ada") == "Hello, Ada!"
