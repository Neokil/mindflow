import { LoaderCircle, X } from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useRef } from "react";
export function Brand({ small = false }: { small?: boolean }) {
  return (
    <span className={`brand ${small ? "small" : ""}`}>
      <span className="brand-mark">
        <svg
          width={small ? 20 : 28}
          height={small ? 20 : 28}
          viewBox="0 0 32 32"
          fill="none"
          aria-hidden="true"
        >
          <path
            d="M16 16C12 9 9 8 6 10s-3 8 0 11 7 1 10-5 7-8 10-6 3 8 0 11-7 1-10-5Z"
            stroke="currentColor"
            strokeWidth="2.5"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      </span>
      mindflow<span className="brand-dot">.</span>
    </span>
  );
}
export function Spinner() {
  return <LoaderCircle className="spin" size={18} aria-label="Loading" />;
}
export function Modal({
  title,
  children,
  close,
  wide = false,
}: {
  title: string;
  children: ReactNode;
  close: () => void;
  wide?: boolean;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const el = dialog.current!;
    el.showModal();
    return () => {
      el.close();
    };
  }, []);
  return (
    <dialog
      ref={dialog}
      className={wide ? "modal wide" : "modal"}
      onCancel={(e) => {
        e.preventDefault();
        close();
      }}
      onClick={(e) => {
        if (e.target === dialog.current) close();
      }}
    >
      <div className="modal-head">
        <h2>{title}</h2>
        <button
          className="icon-button"
          aria-label="Close dialog"
          onClick={close}
        >
          <X size={19} />
        </button>
      </div>
      {children}
    </dialog>
  );
}
export function ErrorMessage({ message }: { message?: string }) {
  return message ? (
    <p className="error" role="alert">
      {message}
    </p>
  ) : null;
}
export function Empty({
  icon,
  title,
  children,
}: {
  icon: ReactNode;
  title: string;
  children: ReactNode;
}) {
  return (
    <div className="empty">
      <div className="empty-icon">{icon}</div>
      <h2>{title}</h2>
      <p>{children}</p>
    </div>
  );
}
