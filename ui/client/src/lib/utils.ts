import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatBytes(bytes: number | null | undefined): string {
  if (bytes == null || bytes < 0) return "—";
  let n = bytes;
  const units = ["B", "KB", "MB", "GB", "TB"];
  for (let i = 0; i < units.length; i++) {
    if (n < 1024 || i === units.length - 1) {
      return i === 0 ? `${Math.round(n)} B` : `${n.toFixed(1)} ${units[i]}`;
    }
    n /= 1024;
  }
  return `${n.toFixed(1)} TB`;
}
