import { words } from "./passphraseWords";
function randomIndex(length: number) {
  const limit = 0x100000000 - (0x100000000 % length);
  const sample = new Uint32Array(1);
  do crypto.getRandomValues(sample);
  while (sample[0] >= limit);
  return sample[0] % length;
}
// Six independently sampled words; rejection sampling avoids modulo bias.
export function suggestPassphrase() {
  const separators = ".!#-+_?=";
  return Array.from({ length: 6 }, (_, index) => {
    const word = words[randomIndex(words.length)];
    return (
      (index ? separators[randomIndex(separators.length)] : "") +
      word[0].toUpperCase() +
      word.slice(1)
    );
  }).join("");
}
