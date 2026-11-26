# deps

A throwaway target for Chalkeus's integration tests: a dependency item (029). The test commits an
approved bump of `requirements.txt` on a bot branch, then names that commit in item 001's spec.
The visible test fails once the bump is applied; the item fixes the code, not the manifest.
