import { Badge } from "@/components/ui/Badge";
import { interventionOutcomeLabels } from "@/lib/format";
import type { InterventionTeam, OutcomeVote } from "@/types";

const voteBadges: Record<OutcomeVote, { label: string; tone: "neutral" | "success" | "danger" }> = {
  "en-attente": { label: "Vote en attente", tone: "neutral" },
  approuve: { label: "A validé", tone: "success" },
  refuse: { label: "A contesté", tone: "danger" },
};

/** Membres de l'équipe et, s'il y en a un, le résultat soumis à leur vote. */
export function TeamMembersList({ team }: { team: InterventionTeam }) {
  return (
    <div className="space-y-4">
      <ul className="space-y-2">
        {team.members.map((member) => (
          <li key={member.hero.id} className="flex flex-wrap items-center gap-2 text-sm">
            <span className={member.isMe ? "font-semibold text-accent" : "font-semibold"}>
              {member.isMe ? "Vous" : member.hero.alias}
            </span>
            {member.role === "lead" && <Badge tone="accent">Lead</Badge>}
            {member.participation === "invite" ? (
              <Badge>Invité, pas encore confirmé</Badge>
            ) : (
              team.proposedOutcome && (
                <Badge tone={voteBadges[member.vote].tone}>{voteBadges[member.vote].label}</Badge>
              )
            )}
          </li>
        ))}
      </ul>

      {team.proposedOutcome && (
        <div className="border-2 border-border bg-background p-3 text-sm">
          <p>
            Résultat proposé
            {team.proposedBy && <> par {team.proposedBy.alias}</>} :{" "}
            <span className="font-semibold">
              {interventionOutcomeLabels[team.proposedOutcome].toLowerCase()}
            </span>
          </p>
          {team.notes && <p className="mt-2 text-pretty text-muted">{team.notes}</p>}
        </div>
      )}
    </div>
  );
}
