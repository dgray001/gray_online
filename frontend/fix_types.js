const { Project } = require("ts-morph");
const fs = require("fs");

const project = new Project({
    tsConfigFilePath: "tsconfig.json",
});

const eslintOutput = JSON.parse(fs.readFileSync("eslint.json", "utf8"));

for (const fileResult of eslintOutput) {
    const filePath = fileResult.filePath;
    const messages = fileResult.messages.filter(m => m.ruleId === "@typescript-eslint/explicit-function-return-type");
    
    if (messages.length === 0) continue;
    
    const sourceFile = project.getSourceFile(filePath);
    if (!sourceFile) continue;
    
    let modified = false;
    
    for (const msg of messages) {
        // Find the node at the specified line and column
        // ESLint line and column are 1-based. ts-morph uses 0-based for some things, but pos is easier.
        // Wait, ts-morph can get node at position.
        // But ESLint column/line might not exactly match the AST node start.
        // Let's just find the function that contains this line.
        
        const line = msg.line;
        // ts-morph gets line number (1-based)
        
        // Find a function, method, or arrow function that starts on this line
        const descendants = sourceFile.getDescendants();
        for (const node of descendants) {
            const kind = node.getKindName();
            if (["FunctionDeclaration", "MethodDeclaration", "ArrowFunction", "FunctionExpression"].includes(kind)) {
                if (node.getStartLineNumber() === line) {
                    if (!node.getReturnTypeNode()) {
                        try {
                            const typeStr = node.getReturnType().getText(node);
                            // Avoid adding weird types like "any" if we can help it, but do it if inferred
                            // Remove import("...") prefixes if any
                            const cleanTypeStr = typeStr.replace(/import\("[^"]+"\)\./g, "");
                            node.setReturnType(cleanTypeStr);
                            modified = true;
                            break; // Move to next message
                        } catch (e) {
                            console.error(`Failed on ${filePath}:${line}`, e);
                        }
                    }
                }
            }
        }
    }
    
    if (modified) {
        sourceFile.saveSync();
    }
}
