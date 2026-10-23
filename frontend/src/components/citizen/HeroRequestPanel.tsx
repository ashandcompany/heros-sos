import { getMyHeroRequest } from "@/lib/api";
import { requireCitizenSession } from "@/lib/server-session";
import { Panel } from "./AccountDetails";
import { HeroRequestForm } from "./HeroRequestForm";

/**
 * « Devenir super-héros » : état de la demande du citoyen, ou formulaire s'il n'en a pas encore
 * déposé (ou si elle a été refusée). Lit la session : à rendre sous <Suspense>.
 */
export async function HeroRequestPanel() {
  const token = await requireCitizenSession();
  const request = await getMyHeroRequest(token);

  return (
    <Panel
      id="hero-request-title"
      title="Devenir super-héros"
      description="Le Bureau examine chaque demande. Une fois acceptée, votre espace super-héros s'ouvre."
    >
      {request?.status === "en-attente" ? (
        <p
          role="status"
          className="rounded-xl border border-rating/40 bg-rating/10 px-4 py-3 text-sm"
        >
          <span className="font-semibold text-rating">Demande en cours d&apos;examen</span> pour le
          pseudo <span className="font-semibold">{request.alias}</span>. Vous serez informé·e de la
          décision ici.
        </p>
      ) : request?.status === "valide" ? (
        <p
          role="status"
          className="rounded-xl border border-success/40 bg-success/10 px-4 py-3 text-sm text-success"
        >
          Votre demande a été acceptée : rechargez la page pour ouvrir votre espace super-héros.
        </p>
      ) : (
        <>
          {request?.status === "rejete" && (
            <div className="mb-6 rounded-xl border border-danger/50 bg-danger/10 px-4 py-3 text-sm">
              <p className="font-semibold text-danger">
                Votre demande pour « {request.alias} » a été refusée.
              </p>
              {request.rejectionReason && <p className="mt-1">Motif : {request.rejectionReason}</p>}
              <p className="mt-1 text-muted">Vous pouvez déposer une nouvelle demande.</p>
            </div>
          )}
          <HeroRequestForm previousAlias={request?.alias} />
        </>
      )}
    </Panel>
  );
}
