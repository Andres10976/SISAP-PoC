import { getKeywordColor } from "./colors";

describe("getKeywordColor", () => {
  it("returns an object with the expected shape", () => {
    const color = getKeywordColor(0);
    expect(color).toHaveProperty("dot");
    expect(color).toHaveProperty("badge");
    expect(color).toHaveProperty("border");
    expect(color).toHaveProperty("activeBg");
    expect(color).toHaveProperty("rowHighlight");
  });

  it("returns the first palette entry for index 0", () => {
    const color = getKeywordColor(0);
    expect(color.dot).toBe("bg-red-400");
  });

  it("returns a different color for index 1", () => {
    const c0 = getKeywordColor(0);
    const c1 = getKeywordColor(1);
    expect(c0.dot).not.toBe(c1.dot);
  });

  it("wraps around with modulo for large indices", () => {
    const c0 = getKeywordColor(0);
    const c8 = getKeywordColor(8);
    expect(c8).toEqual(c0);
  });

  it("handles very large keyword ids without throwing", () => {
    expect(() => getKeywordColor(999999)).not.toThrow();
  });
});
