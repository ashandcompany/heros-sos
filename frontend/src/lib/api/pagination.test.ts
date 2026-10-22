import { describe, expect, it } from "vitest";
import {
  DEFAULT_PAGE_SIZE,
  MAX_PAGE_SIZE,
  fromApiPage,
  pagePosition,
  pageQuery,
  toMockPage,
} from "./pagination";

describe("pageQuery", () => {
  it("ignore les filtres vides et ajoute la pagination par défaut", () => {
    expect(pageQuery({}, { status: "active", q: "", role: undefined, mine: false })).toBe(
      `?status=active&mine=false&limit=${DEFAULT_PAGE_SIZE}&offset=0`,
    );
  });

  it.each([
    [{ limit: 0 }, "limit=1&offset=0"],
    [{ limit: 500 }, `limit=${MAX_PAGE_SIZE}&offset=0`],
    [{ limit: 12.7, offset: 3.9 }, "limit=12&offset=3"],
    [{ limit: Number.NaN, offset: -5 }, `limit=${DEFAULT_PAGE_SIZE}&offset=0`],
  ])("garde %j dans les bornes de l'API", (request, expected) => {
    expect(pageQuery(request)).toBe(`?${expected}`);
  });
});

describe("toMockPage", () => {
  it("découpe la liste comme l'API", () => {
    const items = Array.from({ length: 25 }, (_, index) => index);
    expect(toMockPage(items, { limit: 10, offset: 20 })).toEqual({
      items: [20, 21, 22, 23, 24],
      total: 25,
      limit: 10,
      offset: 20,
    });
  });
});

describe("fromApiPage", () => {
  it("convertit chaque élément et garde la pagination", () => {
    const page = fromApiPage({ items: ["a", "b"], total: 2, limit: 20, offset: 0 }, (item) =>
      item.toUpperCase(),
    );
    expect(page).toEqual({ items: ["A", "B"], total: 2, limit: 20, offset: 0 });
  });
});

describe("pagePosition", () => {
  it("calcule la page courante et le nombre de pages", () => {
    expect(pagePosition({ items: [], total: 45, limit: 20, offset: 40 })).toEqual({
      current: 3,
      count: 3,
    });
  });

  it("compte au moins une page pour une liste vide", () => {
    expect(pagePosition({ items: [], total: 0, limit: 20, offset: 0 }).count).toBe(1);
  });
});
