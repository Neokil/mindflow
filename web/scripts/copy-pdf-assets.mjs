import { cpSync, mkdirSync, rmSync } from "node:fs";
rmSync("dist/pdfjs", { recursive: true, force: true });
mkdirSync("dist/pdfjs", { recursive: true });
for (const directory of ["cmaps", "standard_fonts", "iccs"]) {
  cpSync(`node_modules/pdfjs-dist/${directory}`, `dist/pdfjs/${directory}`, {
    recursive: true,
  });
}
cpSync("node_modules/pdfjs-dist/LICENSE", "dist/pdfjs/LICENSE");
