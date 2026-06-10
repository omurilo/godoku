/**
 * runtimeImport uses native dynamic import so the module is fetched at runtime
 * from the CDN without relying on eval/new Function (CSP-safe). Used for
 * heavy/optional client-only deps like Shiki and Lucide.
 */
export function runtimeImport(url: string): Promise<any> {
  return import(/* @vite-ignore */ url);
}
