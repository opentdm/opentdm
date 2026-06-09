// Transient toast notifications. Provider + useToast() hook (mirrors the
// ColorModeProvider shape). Toasts auto-dismiss; the host is an aria-live region
// so screen readers announce them. Replaces the old inline persistent "Saved"
// text in the editors.
import { createContext, useCallback, useContext, useRef, useState, ReactNode } from "react";
import { CheckCircleIcon, AlertIcon, InfoIcon, XIcon } from "@primer/octicons-react";

export type ToastVariant = "success" | "danger" | "default";

interface Toast {
  id: number;
  variant: ToastVariant;
  message: string;
}

type ToastFn = (message: string, variant?: ToastVariant) => void;

const Ctx = createContext<ToastFn>(() => {});
const DURATION_MS = 3200;

function VariantIcon({ variant }: { variant: ToastVariant }) {
  if (variant === "danger") return <AlertIcon size={16} />;
  if (variant === "default") return <InfoIcon size={16} />;
  return <CheckCircleIcon size={16} />;
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const nextId = useRef(1);

  const remove = useCallback((id: number) => {
    setToasts((ts) => ts.filter((t) => t.id !== id));
  }, []);

  const toast = useCallback<ToastFn>(
    (message, variant = "success") => {
      const id = nextId.current++;
      setToasts((ts) => [...ts, { id, variant, message }]);
      window.setTimeout(() => remove(id), DURATION_MS);
    },
    [remove],
  );

  return (
    <Ctx.Provider value={toast}>
      {children}
      <div className="otdm-toast-host" aria-live="polite" aria-atomic="false">
        {toasts.map((t) => (
          <div key={t.id} className={`otdm-toast otdm-toast-${t.variant}`} role="status">
            <span className="otdm-toast-ico">
              <VariantIcon variant={t.variant} />
            </span>
            <span className="otdm-toast-msg">{t.message}</span>
            <button type="button" className="otdm-toast-x" aria-label="Dismiss" onClick={() => remove(t.id)}>
              <XIcon size={14} />
            </button>
          </div>
        ))}
      </div>
    </Ctx.Provider>
  );
}

export function useToast(): ToastFn {
  return useContext(Ctx);
}
