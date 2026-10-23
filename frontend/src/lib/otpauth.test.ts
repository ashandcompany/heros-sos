import { describe, expect, it } from "vitest";
import { appleOtpauthUrl } from "./otpauth";

describe("appleOtpauthUrl", () => {
  const url =
    "otpauth://totp/Heros-SOS:fury%40shield.example?secret=JBSWY3DPEHPK3PXP&issuer=Heros-SOS&algorithm=SHA1&digits=6&period=30";

  it("sans domaine, ne change que le schéma : hôte, étiquette et paramètres (dont le secret) restent identiques", () => {
    expect(appleOtpauthUrl(url)).toBe(url.replace("otpauth:", "apple-otpauth:"));
  });

  it("avec un domaine, remplace l'émetteur et garde l'étiquette et tous les autres paramètres", () => {
    const apple = new URL(appleOtpauthUrl(url, "heros-sos.example"));
    expect(apple.protocol).toBe("apple-otpauth:");
    expect(apple.host).toBe("totp");
    expect(apple.pathname).toBe("/Heros-SOS:fury%40shield.example");
    expect(Object.fromEntries(apple.searchParams)).toEqual({
      secret: "JBSWY3DPEHPK3PXP",
      issuer: "heros-sos.example",
      algorithm: "SHA1",
      digits: "6",
      period: "30",
    });
  });

  it("ajoute l'émetteur s'il manquait", () => {
    const apple = new URL(appleOtpauthUrl("otpauth://totp/X:y?secret=ABC", "localhost"));
    expect(apple.searchParams.get("issuer")).toBe("localhost");
  });

  it("laisse intacte une adresse qui n'est pas du schéma otpauth", () => {
    expect(appleOtpauthUrl("https://example.org/", "x.example")).toBe("https://example.org/");
    expect(appleOtpauthUrl("apple-otpauth://totp/X:y?secret=ABC")).toBe(
      "apple-otpauth://totp/X:y?secret=ABC",
    );
  });
});
