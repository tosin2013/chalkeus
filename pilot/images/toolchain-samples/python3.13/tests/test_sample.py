import yaml


def test_yaml_from_the_cache():
    assert yaml.safe_load("a: 1") == {"a": 1}
