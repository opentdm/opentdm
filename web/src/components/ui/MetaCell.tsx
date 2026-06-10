import { CSSProperties, ReactNode } from "react";

interface MetaCellProps {
  label: string;
  children: ReactNode;
  valueStyle?: CSSProperties;
}

// One labelled cell of the project meta grid (.otdm-meta-grid). A small `k`
// label over a large `v` value; valueStyle tweaks the value (e.g. the role cell).
export function MetaCell({ label, children, valueStyle }: MetaCellProps) {
  return (
    <div className="otdm-meta-cell">
      <div className="k">{label}</div>
      <div className="v" style={valueStyle}>
        {children}
      </div>
    </div>
  );
}
