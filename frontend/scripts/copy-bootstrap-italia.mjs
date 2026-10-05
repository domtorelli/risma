// Copia font e icone di Bootstrap Italia in public/, da dove vengono serviti
// come file statici. Gira prima di "dev" e "build": i file copiati non sono
// nel repository, la fonte resta il pacchetto npm alla versione installata.
import { cpSync, rmSync } from "node:fs";

const source = "node_modules/bootstrap-italia/dist";
const target = "public/bootstrap-italia";

rmSync(target, { recursive: true, force: true });
cpSync(`${source}/fonts`, `${target}/fonts`, { recursive: true });
cpSync(`${source}/svg/sprites.svg`, `${target}/svg/sprites.svg`);

console.log(`Bootstrap Italia: font e icone copiati in ${target}`);
