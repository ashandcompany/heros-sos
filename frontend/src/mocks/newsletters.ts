import type { NewsletterIssue } from "@/types";

export const newslettersMock: NewsletterIssue[] = [
  {
    id: "newsletter-2",
    subject: "Les nouveaux héros de l'automne",
    body: "<p>Brouillon : présenter les héros arrivés ce trimestre.</p>",
    recipientUserId: null,
    createdAt: "2026-10-20T09:30:00+02:00",
    sentAt: null,
  },
  {
    id: "newsletter-1",
    subject: "Heros-SOS : le bilan de la rentrée",
    body: "<p>Bonjour,</p><p>En septembre, <strong>142 incidents</strong> ont été résolus grâce à vos signalements.</p><h2>À retenir</h2><ul><li><p>La carte des incidents est désormais en temps réel.</p></li><li><p>Docteur Rouille a été aperçu près des quais : restez vigilants.</p></li></ul><p>Merci de faire vivre le réseau !</p>",
    recipientUserId: null,
    createdAt: "2026-10-01T08:00:00+02:00",
    sentAt: "2026-10-01T10:15:00+02:00",
  },
];
