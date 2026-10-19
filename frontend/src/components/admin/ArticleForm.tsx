"use client";

import { useActionState } from "react";
import { saveArticleAction } from "@/app/actions/admin";
import { FormMessage } from "@/components/ui/FormMessage";
import { RadioGroup } from "@/components/ui/RadioGroup";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextAreaField } from "@/components/ui/TextAreaField";
import { TextField } from "@/components/ui/TextField";
import {
  ARTICLE_EXCERPT_MAX_LENGTH,
  ARTICLE_TITLE_MAX_LENGTH,
  articleStatusOptions,
  type ArticleField,
} from "@/lib/validation/admin";
import { idleFormState, type FormState } from "@/lib/validation/form-state";
import type { BlogArticle } from "@/types";
import { RichTextEditor } from "./RichTextEditor";

/** Création (sans `article`) ou modification d'un article du blog. */
export function ArticleForm({ article }: { article?: BlogArticle }) {
  const [state, formAction] = useActionState(
    saveArticleAction,
    idleFormState as FormState<ArticleField>,
  );
  const errors = state.fieldErrors;
  const value = (field: ArticleField, saved: string | undefined) =>
    state.values?.[field] ?? saved ?? "";

  return (
    <form action={formAction} noValidate className="space-y-6">
      {article && <input type="hidden" name="id" value={article.id} />}

      <TextField
        label="Titre"
        name="title"
        required
        maxLength={ARTICLE_TITLE_MAX_LENGTH}
        defaultValue={value("title", article?.title)}
        error={errors.title}
      />

      <TextAreaField
        label="Chapeau"
        hint={`Résumé affiché dans la liste des articles (${ARTICLE_EXCERPT_MAX_LENGTH} caractères maximum).`}
        name="excerpt"
        rows={2}
        required
        maxLength={ARTICLE_EXCERPT_MAX_LENGTH}
        defaultValue={value("excerpt", article?.excerpt)}
        error={errors.excerpt}
      />

      {/* L'éditeur garde sa saisie après l'envoi (pas de réinitialisation, contrairement aux champs natifs). */}
      <RichTextEditor
        label="Contenu"
        name="content"
        defaultValue={article?.content ?? ""}
        error={errors.content}
      />

      <RadioGroup
        legend="Statut"
        name="status"
        options={articleStatusOptions}
        defaultValue={value("status", article?.status ?? "brouillon")}
        error={errors.status}
        required
      />

      <FormMessage state={state} />
      <SubmitButton pendingLabel="Enregistrement…">
        {article ? "Enregistrer l'article" : "Créer l'article"}
      </SubmitButton>
    </form>
  );
}
