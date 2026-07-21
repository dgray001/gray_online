declare module '*.html' {
  const value: string;
  export default value;
}

declare module '*.png' {
  const value: unknown;
  export = value;
}

declare module '*.scss';

declare module 'eslint-config-google';
