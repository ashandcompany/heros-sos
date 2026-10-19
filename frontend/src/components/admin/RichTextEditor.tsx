"use client";

import { EditorContent, useEditor, useEditorState, type Editor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { useId, useState, type ReactNode } from "react";
import { cn } from "@/components/ui/cn";

interface ToolbarButtonProps {
  label: string;
  onClick: () => void;
  active?: boolean;
  disabled?: boolean;
  children: ReactNode;
}

function ToolbarButton({ label, onClick, active, disabled, children }: ToolbarButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={label}
      title={label}
      aria-pressed={active === undefined ? undefined : active}
      className={cn(
        "inline-flex size-10 items-center justify-center rounded-md font-mono text-sm font-semibold disabled:opacity-40",
        active ? "bg-primary text-primary-foreground" : "text-foreground hover:bg-surface-raised",
      )}
    >
      {children}
    </button>
  );
}

function Toolbar({ editor }: { editor: Editor }) {
  const state = useEditorState({
    editor,
    selector: ({ editor: current }) => ({
      bold: current.isActive("bold"),
      italic: current.isActive("italic"),
      underline: current.isActive("underline"),
      h2: current.isActive("heading", { level: 2 }),
      h3: current.isActive("heading", { level: 3 }),
      bulletList: current.isActive("bulletList"),
      orderedList: current.isActive("orderedList"),
      blockquote: current.isActive("blockquote"),
      link: current.isActive("link"),
      canUndo: current.can().undo(),
      canRedo: current.can().redo(),
    }),
  });
  const chain = () => editor.chain().focus();

  const toggleLink = () => {
    if (state.link) {
      chain().extendMarkRange("link").unsetLink().run();
      return;
    }
    const url = window.prompt("Adresse du lien (https://…)")?.trim();
    if (url) chain().extendMarkRange("link").setLink({ href: url }).run();
  };

  return (
    <div
      role="toolbar"
      aria-label="Mise en forme"
      className="flex flex-wrap gap-1 border-b border-border bg-surface p-1.5"
    >
      <ToolbarButton label="Gras" active={state.bold} onClick={() => chain().toggleBold().run()}>
        B
      </ToolbarButton>
      <ToolbarButton
        label="Italique"
        active={state.italic}
        onClick={() => chain().toggleItalic().run()}
      >
        <span className="italic">I</span>
      </ToolbarButton>
      <ToolbarButton
        label="Souligné"
        active={state.underline}
        onClick={() => chain().toggleUnderline().run()}
      >
        <span className="underline">S</span>
      </ToolbarButton>
      <span aria-hidden="true" className="mx-1 w-px self-stretch bg-border" />
      <ToolbarButton
        label="Intertitre"
        active={state.h2}
        onClick={() => chain().toggleHeading({ level: 2 }).run()}
      >
        H2
      </ToolbarButton>
      <ToolbarButton
        label="Sous-intertitre"
        active={state.h3}
        onClick={() => chain().toggleHeading({ level: 3 }).run()}
      >
        H3
      </ToolbarButton>
      <ToolbarButton
        label="Liste à puces"
        active={state.bulletList}
        onClick={() => chain().toggleBulletList().run()}
      >
        •≡
      </ToolbarButton>
      <ToolbarButton
        label="Liste numérotée"
        active={state.orderedList}
        onClick={() => chain().toggleOrderedList().run()}
      >
        1.
      </ToolbarButton>
      <ToolbarButton
        label="Citation"
        active={state.blockquote}
        onClick={() => chain().toggleBlockquote().run()}
      >
        “”
      </ToolbarButton>
      <ToolbarButton
        label={state.link ? "Retirer le lien" : "Ajouter un lien"}
        active={state.link}
        onClick={toggleLink}
      >
        🔗
      </ToolbarButton>
      <span aria-hidden="true" className="mx-1 w-px self-stretch bg-border" />
      <ToolbarButton label="Annuler" disabled={!state.canUndo} onClick={() => chain().undo().run()}>
        ↶
      </ToolbarButton>
      <ToolbarButton
        label="Rétablir"
        disabled={!state.canRedo}
        onClick={() => chain().redo().run()}
      >
        ↷
      </ToolbarButton>
    </div>
  );
}

interface RichTextEditorProps {
  label: string;
  /** Nom du champ caché qui transmet le HTML au formulaire. */
  name: string;
  defaultValue: string;
  hint?: string;
  error?: string;
}

/**
 * Éditeur de texte riche (Tiptap) pour les articles du blog. Le HTML produit est recopié dans un
 * champ caché : le formulaire parent l'envoie comme n'importe quel autre champ.
 */
export function RichTextEditor({ label, name, defaultValue, hint, error }: RichTextEditorProps) {
  const id = useId();
  const labelId = `${id}-label`;
  const hintId = `${id}-hint`;
  const errorId = `${id}-error`;
  const describedBy = [hint && hintId, error && errorId].filter(Boolean).join(" ");
  const [html, setHtml] = useState(defaultValue);

  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: { levels: [2, 3] },
        code: false,
        codeBlock: false,
        link: {
          openOnClick: false,
          defaultProtocol: "https",
          protocols: ["http", "https", "mailto"],
          HTMLAttributes: { rel: "noopener noreferrer nofollow", target: null },
        },
      }),
    ],
    content: defaultValue,
    // Rendu côté serveur : l'éditeur n'est créé qu'une fois dans le navigateur.
    immediatelyRender: false,
    editorProps: {
      attributes: {
        role: "textbox",
        "aria-multiline": "true",
        "aria-labelledby": labelId,
        ...(describedBy && { "aria-describedby": describedBy }),
        ...(error && { "aria-invalid": "true" }),
        class: cn(
          "min-h-80 px-4 py-3 focus:outline-none",
          "[&_a]:text-accent [&_a]:underline [&_blockquote]:border-l-2 [&_blockquote]:border-accent [&_blockquote]:pl-4 [&_blockquote]:italic",
          "[&_h2]:mt-6 [&_h2]:font-display [&_h2]:text-xl [&_h2]:font-bold [&_h3]:mt-4 [&_h3]:font-display [&_h3]:text-lg [&_h3]:font-semibold",
          "[&_ol]:list-decimal [&_ol]:pl-6 [&_ul]:list-disc [&_ul]:pl-6 [&_li]:my-1 [&_p]:my-3",
        ),
      },
    },
    onUpdate: ({ editor: current }) => setHtml(current.isEmpty ? "" : current.getHTML()),
  });

  return (
    <div>
      <p id={labelId} className="mb-2 text-sm font-medium">
        {label}
      </p>
      <input type="hidden" name={name} value={html} />
      <div
        className={cn(
          "overflow-hidden rounded-xl border bg-background has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-primary",
          error ? "border-danger" : "border-border",
        )}
      >
        {editor ? (
          <>
            <Toolbar editor={editor} />
            <EditorContent editor={editor} />
          </>
        ) : (
          <div aria-hidden="true" className="h-[23rem] motion-safe:animate-pulse" />
        )}
      </div>
      {hint && (
        <p id={hintId} className="mt-2 text-sm text-muted">
          {hint}
        </p>
      )}
      {error && (
        <p id={errorId} className="mt-2 text-sm text-danger">
          {error}
        </p>
      )}
    </div>
  );
}
