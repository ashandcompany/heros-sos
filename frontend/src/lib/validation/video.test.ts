import { describe, expect, it } from "vitest";
import {
  VIDEO_DESCRIPTION_MAX_LENGTH,
  VIDEO_TITLE_MAX_LENGTH,
  readFreeVideoValues,
  validateFreeVideo,
} from "./video";

const video = {
  title: "Le sauvetage du pont",
  description: "",
  videoUrl: "https://video.example/1",
};

describe("validateFreeVideo", () => {
  it("accepte une vidéo complète", () => {
    expect(Object.values(validateFreeVideo(video)).filter(Boolean)).toEqual([]);
  });

  it.each([
    ["https://video.example/1", true],
    ["http://video.example/1", true],
    ["video.example/1", false],
    ["ftp://video.example/1", false],
    ["javascript:alert(1)", false],
    ["", false],
  ])("lien %s valide : %s", (videoUrl, valid) => {
    expect(validateFreeVideo({ ...video, videoUrl }).videoUrl === undefined).toBe(valid);
  });

  it("exige un titre et borne le titre et la description", () => {
    expect(validateFreeVideo({ ...video, title: "" }).title).toBeDefined();
    expect(
      validateFreeVideo({ ...video, title: "a".repeat(VIDEO_TITLE_MAX_LENGTH + 1) }).title,
    ).toBeDefined();
    expect(
      validateFreeVideo({ ...video, description: "a".repeat(VIDEO_DESCRIPTION_MAX_LENGTH + 1) })
        .description,
    ).toBeDefined();
  });
});

describe("readFreeVideoValues", () => {
  it("lit et nettoie les champs du formulaire", () => {
    const formData = new FormData();
    formData.set("title", "  Titre ");
    formData.set("videoUrl", " https://video.example/1 ");
    expect(readFreeVideoValues(formData)).toEqual({
      title: "Titre",
      description: "",
      videoUrl: "https://video.example/1",
    });
  });
});
