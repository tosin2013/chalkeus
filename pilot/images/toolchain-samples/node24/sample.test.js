const test = require('node:test'); const assert = require('node:assert'); const semver = require('semver');
test('semver from the cache', () => { assert.ok(semver.gt('1.2.4', '1.2.3')); });
