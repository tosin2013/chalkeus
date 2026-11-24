"""Image build arguments from pilot/pins.env (031: per-architecture pins, contracts/pins.md)."""
import importlib.util
import unittest
from pathlib import Path

_path = Path(__file__).resolve().parents[1] / "build_args.py"
_spec = importlib.util.spec_from_file_location("build_args", _path)
ba = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(ba)

IMAGES = ["worker", "tools", "egress", "toolchain-python3.13", "toolchain-node24", "toolchain-go",
          "toolchain-jdk21-maven"]

# The per-arch keys each image needs (031); every one has an _AMD64 and an _ARM64 value.
PER_ARCH = {
    "worker": ["OPENCODE_ASSET", "OPENCODE_ASSET_SHA256", "OPENCODE_SHA256"],
    "tools": ["SPEC_ASSET", "SPEC_ASSET_SHA256", "SPEC_SHA256"],
    "egress": [],
    "toolchain-python3.13": ["PYTHON313_ASSET", "PYTHON313_ASSET_SHA256"],
    "toolchain-node24": ["NODE24_SHA256"],
    "toolchain-go": ["GO_SHA256"],
    "toolchain-jdk21-maven": ["TEMURIN21_SHA256"],
}

REMOVED = ["OPENCODE_ASSET", "OPENCODE_ASSET_SHA256", "OPENCODE_SHA256", "SPEC_ASSET", "SPEC_ASSET_SHA256",
           "SPEC_SHA256", "PYTHON313_ASSET", "PYTHON313_ASSET_SHA256"]


class TestBuildArgs(unittest.TestCase):
    def setUp(self):
        self.pins = ba.read_pins()

    def test_every_image_has_both_architectures(self):
        for image in IMAGES:
            args = ba.build_args(image, self.pins)
            for key in PER_ARCH[image]:
                for arch in ("AMD64", "ARM64"):
                    with self.subTest(image=image, key=f"{key}_{arch}"):
                        self.assertTrue(args.get(f"{key}_{arch}"))

    def test_checksums_are_sha256(self):
        for image in IMAGES:
            for k, v in ba.build_args(image, self.pins).items():
                if "SHA256" in k:
                    with self.subTest(image=image, key=k):
                        self.assertRegex(v, r"^[0-9a-f]{64}$")

    def test_architectures_differ(self):
        # A copy-paste of the amd64 value into the arm64 key would pass every other check.
        for image in IMAGES:
            args = ba.build_args(image, self.pins)
            for key in PER_ARCH[image]:
                with self.subTest(image=image, key=key):
                    self.assertNotEqual(args[f"{key}_AMD64"], args[f"{key}_ARM64"])

    def test_old_single_arch_keys_are_gone(self):
        for key in REMOVED:
            with self.subTest(key=key):
                self.assertNotIn(key, self.pins)

    def test_missing_pin_fails_naming_it(self):
        for image in IMAGES:
            for key in ba.build_args(image, self.pins):
                if key not in self.pins:
                    continue  # derived values (repo, tag, checksum file name)
                pins = dict(self.pins)
                del pins[key]
                with self.subTest(image=image, key=key), self.assertRaises(SystemExit) as cm:
                    ba.build_args(image, pins)
                self.assertEqual(str(cm.exception.code), f"pins.env: missing {key}")

    def test_unknown_image(self):
        with self.assertRaises(SystemExit):
            ba.build_args("nope", self.pins)


if __name__ == "__main__":
    unittest.main()
