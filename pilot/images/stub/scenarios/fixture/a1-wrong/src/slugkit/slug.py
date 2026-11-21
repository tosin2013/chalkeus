"""A deliberately incomplete slugify (the stub's induced A/1): it lowercases and joins words, but
ignores the rules in the withheld acceptance details."""
SEPARATOR = "-"
SLUG_MAX = 1000
EMPTY_SLUG = ""


def slugify(text: str) -> str:
    return SEPARATOR.join(text.lower().split())
