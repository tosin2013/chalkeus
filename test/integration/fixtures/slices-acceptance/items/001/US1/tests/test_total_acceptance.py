from stats.total import total


def test_adds_numbers():
    assert total([1, 2, 3]) == 6
