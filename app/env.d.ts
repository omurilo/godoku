/*
 * Lightweight ambient type shim.
 *
 * GoDoku has no node_modules (the React shell is bundled from esm.sh at build
 * time and esbuild strips types without type-checking). These declarations stop
 * the editor's language server from reporting "Cannot find module 'react'" and
 * missing-JSX errors. They are intentionally loose (everything is `any`) — for
 * full IntelliSense, run `npm i -D @types/react @types/react-dom` in this
 * folder and delete this file.
 */

declare module "react" {
  namespace React {
    type ReactNode = any;
    type ComponentType<P = any> = (props: P) => any;
    type FC<P = any> = (props: P) => any;
    type ButtonHTMLAttributes<T = any> = any;
    type HTMLAttributes<T = any> = any;
    type AnchorHTMLAttributes<T = any> = any;
    type KeyboardEvent<T = any> = any;
    type MouseEvent<T = any> = any;
    type ChangeEvent<T = any> = any;
    type RefObject<T = any> = any;

    const createElement: any;
    const Fragment: any;
    function forwardRef<T = any, P = any>(render: any): any;
    function createContext<T = any>(defaultValue?: any): any;
    function useState<T = any>(initial?: any): [any, any];
    function useEffect(effect: any, deps?: any): void;
    function useCallback<T = any>(fn: T, deps?: any): T;
    function useContext<T = any>(ctx: any): any;
    function useRef<T = any>(initial?: any): any;
    function useMemo<T = any>(fn: any, deps?: any): any;

    namespace JSX {
      interface IntrinsicElements {
        [elemName: string]: any;
      }
      interface ElementChildrenAttribute {
        children: any;
      }
      interface IntrinsicAttributes {
        key?: any;
      }
      type Element = any;
      type ElementClass = any;
    }
  }
  export = React;
}

declare module "react-dom" {
  export const createPortal: any;
}
declare module "react-dom/client" {
  export const hydrateRoot: any;
  export const createRoot: any;
}
declare module "react-dom/server" {
  export const renderToString: any;
  export const renderToStaticMarkup: any;
}
declare module "clsx" {
  export type ClassValue = any;
  export const clsx: any;
  const _default: any;
  export default _default;
}
declare module "tailwind-merge" {
  export const twMerge: any;
}
declare module "highlight.js/lib/common" {
  const hljs: any;
  export default hljs;
}

declare namespace JSX {
  interface IntrinsicElements {
    [elemName: string]: any;
  }
  interface ElementChildrenAttribute {
    children: any;
  }
  interface IntrinsicAttributes {
    key?: any;
  }
  type Element = any;
}

interface Window {
  __GODOKU_PROPS__?: Record<string, unknown>;
}
