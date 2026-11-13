import { expect, it } from "vitest";
import { suggestPassphrase } from "./passphrase";
import { words } from "./passphraseWords";
it("suggests six English words separated by special characters within password limits", () => {
  const values = new Set(Array.from({ length: 20 }, suggestPassphrase));
  expect(values.size).toBe(20);
  for (const value of values) {
    expect(value.length).toBeLessThanOrEqual(128);
    const selected = value.split(/[.!#\-+_?=]/);
    expect(selected).toHaveLength(6);
    for (const word of selected) expect(words).toContain(word.toLowerCase());
  }
});
