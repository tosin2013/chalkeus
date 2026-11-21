from slugkit import slugify
from slugkit.slug import SEPARATOR


def test_slugify_joins_words_with_the_separator():
    assert slugify("Hello World") == f"hello{SEPARATOR}world"
