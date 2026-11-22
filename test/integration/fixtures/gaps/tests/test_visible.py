"""Visible tests. One fails on purpose: doctor's baseline must refuse this target (025 T013)."""


def test_package_imports():
    import gaps
    assert gaps.__doc__


def test_broken_on_purpose():
    assert 1 + 1 == 3, "this target's own tests fail, so doctor --baseline fails"
