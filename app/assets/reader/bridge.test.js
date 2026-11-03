const test = require('node:test');
const assert = require('node:assert');
const { handle } = require('./bridge.js');

test('next calls next', () => {
  const view = { next: () => { view.called = true; }, prev(){}, goTo(){}, setStyle(){} };
  handle({ type: 'next' }, view);
  assert.ok(view.called);
});
