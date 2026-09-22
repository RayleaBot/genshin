// Helpers shared by the calculation bundlers.
import fs from 'node:fs/promises'
import path from 'node:path'

export async function walk(directory) {
  const files = []
  for (const item of await fs.readdir(directory, { withFileTypes: true })) {
    const full = path.join(directory, item.name)
    if (item.isDirectory()) files.push(...await walk(full))
    else files.push(full)
  }
  return files.sort()
}

// characters/<id>-<name>.js, keeping only letters and digits of the name:
// Go embeds reject punctuation such as the middle dot in 奇偶·男性.
export function calcScriptName(key, name) {
  const readable = [...name].filter(ch => /[\p{L}\p{N}]/u.test(ch)).join('')
  return 'characters/' + key.slice(key.indexOf('_') + 1) + '-' + readable + '.js'
}

// Transpile one upstream module to a self-contained CommonJS expression. Any
// import outside allowedImports stops the bundle, so no unreviewed code enters.
export function transpile(typescript, file, source, allowedImports) {
  const ast = typescript.createSourceFile(file, source, typescript.ScriptTarget.Latest, true)
  for (const statement of ast.statements) {
    if (typescript.isImportDeclaration(statement) && !allowedImports.includes(statement.moduleSpecifier.text)) {
      throw new Error('Unapproved calculation import: ' + file + ' ' + statement.moduleSpecifier.text)
    }
  }
  const built = typescript.transpileModule(source.replace(/^import .*$/gm, ''), {
    compilerOptions: { target: typescript.ScriptTarget.ES2019, module: typescript.ModuleKind.CommonJS },
    reportDiagnostics: true,
  })
  if (built.diagnostics.some(d => d.category === typescript.DiagnosticCategory.Error)) throw new Error('Calculation syntax: ' + file)
  return `(()=>{const exports={};\n${built.outputText}\nreturn exports;})()`
}
