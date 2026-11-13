import { expect, it } from "vitest";
import { retainCheckedLines } from "./textLists";
it("retains checked items when lines are inserted and removed", () => {
  expect(
    retainCheckedLines("First\nSecond", "New\nFirst\nSecond", [1]),
  ).toEqual([2]);
  expect(retainCheckedLines("First\nSecond", "Second", [1])).toEqual([0]);
  expect(retainCheckedLines("Same\nSame", "Same\nSame", [1])).toEqual([1]);
});
