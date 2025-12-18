import google from 'eslint-config-google';
import prettier from 'eslint-config-prettier';
import tsEslintPlugin from '@typescript-eslint/eslint-plugin';
import tsEslintParser from '@typescript-eslint/parser';

// The eslint-config-google package is trying to use these deprecated rules
// Must be before loading the google package
if (google.rules) {
	delete google.rules['valid-jsdoc'];
	delete google.rules['require-jsdoc'];
}

export default [
	google,
	prettier,
	{
		files: ["src/**/*.{js,mjs,ts,tsx}"],
		rules: {
			"no-unused-vars": "off",
		}
	},
	{
		files: ["src/**/*.{ts,tsx}"],
		plugins: {
			"@typescript-eslint": tsEslintPlugin
		},
		languageOptions: {
			parser: tsEslintParser,
			parserOptions: {
				project: './tsconfig.json'
			}
		},
		rules: {
			"eqeqeq": ["error", "always"],
			"prefer-const": "error",
			"block-scoped-var": "error",
			"no-use-before-define": ["error", { "functions": false, "classes": false, "variables": true }],
			"no-console": "off",
			"no-alert": "off",
			"@typescript-eslint/consistent-type-imports": ["error", { "prefer": "type-imports" }],
			"@typescript-eslint/no-explicit-any": "error",
			"camelcase": "off",
			"@typescript-eslint/naming-convention": [
				"error",
				// Enforce PascalCase for Classes and Interfaces
				{
					selector: ["class", "interface"],
					format: ["PascalCase"],
				},
				// Enforce camelCase for functions and methods
				{
					selector: ["function", "method"],
					format: ["camelCase"],
					leadingUnderscore: "allow",
				},
				// Enforce snake_case for standard variables and properties
				{
					selector: ["variableLike", "property", "parameter"],
					format: ["snake_case", "UPPER_CASE"],
					leadingUnderscore: "allow",
				},
				// Allow registering custom elements
				{
					selector: ["objectLiteralProperty", "typeProperty"],
					format: null,
					filter: {
						regex: "^dwg-[a-z-]+$",
						match: true,
					},
				},
				// Allow headers objects to have Header-Case names
				{
					selector: ["objectLiteralProperty", "typeProperty"],
					format: null,
					filter: {
						regex: "^(Accept|Content-Type|Authorization|User-Agent|Cache-Control|If-None-Match|X-[-_a-zA-Z0-9]+)$",
						match: true,
					},
				},
			],
			"@typescript-eslint/no-unused-vars": [
				"warn",
				{
					// This regex ignores any variable that starts with an underscore
					"argsIgnorePattern": "^_",
					"varsIgnorePattern": "^_", // Also ignore local variables starting with _
					"caughtErrorsIgnorePattern": "^_" // Ignore catch block variables like `catch (_e)`
				}
			],
		},
		settings: {},
	},
];