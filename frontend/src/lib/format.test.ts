import { describe, expect, it } from "vitest";
import { formatFullName, formatLocation, formatPhone, formatRetryDelay, phoneHref } from "./format";

describe("formatPhone et phoneHref", () => {
  it.each([
    ["0612345678", "06 12 34 56 78", "tel:+33612345678"],
    ["+33612345678", "+33 6 12 34 56 78", "tel:+33612345678"],
    ["+4915112345678", "+4915112345678", "tel:+4915112345678"],
  ])("%s", (phone, formatted, href) => {
    expect(formatPhone(phone)).toBe(formatted);
    expect(phoneHref(phone)).toBe(href);
  });
});

describe("formatFullName", () => {
  it("met le nom en capitales devant le prénom", () => {
    expect(formatFullName("Dupont", "Camille")).toBe("DUPONT Camille");
    expect(formatFullName(null, "Camille")).toBe("Camille");
    expect(formatFullName(null, null)).toBeNull();
  });
});

describe("formatRetryDelay", () => {
  it("arrondit à la minute supérieure, une minute par défaut", () => {
    expect(formatRetryDelay(undefined)).toBe("1 minute");
    expect(formatRetryDelay(61)).toBe("2 minutes");
  });
});

describe("formatLocation", () => {
  it("ajoute le quartier quand il est connu", () => {
    expect(formatLocation({ city: "Paris", district: "Marais" })).toBe("Paris · Marais");
    expect(formatLocation({ city: "Paris" })).toBe("Paris");
  });
});
