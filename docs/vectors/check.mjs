import fs from 'fs';
const files = fs.readdirSync('docs/vectors').filter(f=>f.endsWith('.json'));
console.log('vectors:', files.length);
for (const f of files) JSON.parse(fs.readFileSync('docs/vectors/'+f,'utf8'));
console.log('ok');
