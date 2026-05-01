export function isStaticExport(): boolean {
  return import.meta.env.VITE_STATIC_EXPORT === '1';
}
