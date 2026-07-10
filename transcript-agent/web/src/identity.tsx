import { createContext, useContext, useState, type ReactNode } from "react";

export interface Identity {
  userId: string;
  role: "producer" | "reviewer" | "admin";
}

export const IDENTITIES: Identity[] = [
  { userId: "producer-1", role: "producer" },
  { userId: "reviewer-1", role: "reviewer" },
  { userId: "admin-1", role: "admin" },
];

const STORAGE_KEY = "transcript-agent.identity";

export function getStoredIdentity(): Identity {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as Identity;
      const match = IDENTITIES.find((i) => i.userId === parsed.userId);
      if (match) return match;
    }
  } catch {
    // fall through to default
  }
  return IDENTITIES[0];
}

export function storeIdentity(identity: Identity): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(identity));
}

interface IdentityContextValue {
  identity: Identity;
  setIdentity: (identity: Identity) => void;
}

const IdentityContext = createContext<IdentityContextValue | null>(null);

export function IdentityProvider({
  children,
  onChange,
}: {
  children: ReactNode;
  onChange?: (identity: Identity) => void;
}) {
  const [identity, setIdentityState] = useState<Identity>(getStoredIdentity);

  const setIdentity = (next: Identity) => {
    storeIdentity(next);
    setIdentityState(next);
    onChange?.(next);
  };

  return (
    <IdentityContext.Provider value={{ identity, setIdentity }}>
      {children}
    </IdentityContext.Provider>
  );
}

export function useIdentity(): IdentityContextValue {
  const ctx = useContext(IdentityContext);
  if (!ctx) throw new Error("useIdentity must be used within IdentityProvider");
  return ctx;
}

export function canApprove(identity: Identity): boolean {
  return identity.role === "reviewer" || identity.role === "admin";
}

/**
 * UX mirrors of the server-side role rules (PRD 16.2). The server remains the
 * real enforcer — these only hide/disable controls that would 403 anyway.
 */
export function canGenerateExports(identity: Identity): boolean {
  return identity.role === "reviewer" || identity.role === "admin";
}

export function canReopen(identity: Identity): boolean {
  return identity.role === "reviewer" || identity.role === "admin";
}

export function canCancel(identity: Identity, submittedBy: string): boolean {
  return identity.role === "admin" || identity.userId === submittedBy;
}
