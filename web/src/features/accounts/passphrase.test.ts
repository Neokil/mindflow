import { expect, it } from "vitest";
import { suggestPassphrase } from "./passphrase";
import { words } from "./passphraseWords";
it("suggests two or three English words with punctuation within 10–15 characters", () => {
  const values = Array.from({ length: 100 }, suggestPassphrase);
  for (const value of values) {
    expect(value.length).toBeGreaterThanOrEqual(10);
    expect(value.length).toBeLessThanOrEqual(15);
    expect(value).toMatch(/^[A-Z][a-z]+(?:[.!#\-+_?=][A-Z][a-z]+){1,2}$/);
    const selected = value.split(/[.!#\-+_?=]/);
    expect(selected.length).toBeGreaterThanOrEqual(2);
    expect(selected.length).toBeLessThanOrEqual(3);
    for (const word of selected) expect(words).toContain(word.toLowerCase());
  }
});
