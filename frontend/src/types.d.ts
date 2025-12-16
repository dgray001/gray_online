declare module '*.html' {
  const value: string;
  export default value;
}

declare module '*.png' {
  const value: unknown;
  export = value;
}

declare module 'eslint-config-google';

// TODO: Add custom events here

// Custom headers
declare global {
  interface Headers {
    'X-File-Name': string;
  }
}

export {};