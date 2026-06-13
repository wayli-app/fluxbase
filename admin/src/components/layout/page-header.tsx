import type { ReactNode } from "react";

interface PageHeaderProps {
  title: ReactNode;
  description?: ReactNode;
  icon?: ReactNode;
  actions?: ReactNode;
  leading?: ReactNode;
}

export function PageHeader({
  title,
  description,
  icon,
  actions,
  leading,
}: PageHeaderProps) {
  return (
    <div className="bg-background flex items-center justify-between border-b px-6 py-4">
      <div className="flex items-center gap-3">
        {leading}
        {icon && (
          <div className="bg-primary/10 flex h-10 w-10 items-center justify-center rounded-lg">
            <span className="text-primary inline-flex h-5 w-5 items-center justify-center [&>svg]:h-5 [&>svg]:w-5">
              {icon}
            </span>
          </div>
        )}
        <div>
          <h1 className="text-xl font-semibold">{title}</h1>
          {description && (
            <p className="text-muted-foreground text-sm">{description}</p>
          )}
        </div>
      </div>
      {actions && <div className="flex items-center gap-2">{actions}</div>}
    </div>
  );
}
