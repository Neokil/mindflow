import { words } from "./passphraseWords";
function randomIndex(length: number) {
  const limit = 0x100000000 - (0x100000000 % length);
  const sample = new Uint32Array(1);
  do crypto.getRandomValues(sample);
  while (sample[0] >= limit);
  return sample[0] % length;
}
// Sample whole words again when punctuation would put the result outside 10–15 characters.
export function suggestPassphrase() {
  const separators = ".!#-+_?=";
  const count = 2 + randomIndex(2);
  let password: string;
  do {
    password = Array.from({ length: count }, (_, index) => {
      const word = words[randomIndex(words.length)];
      return (
        (index ? separators[randomIndex(separators.length)] : "") +
        word[0].toUpperCase() +
        word.slice(1)
      );
    }).join("");
  } while (password.length < 10 || password.length > 15);
  return password;
}
