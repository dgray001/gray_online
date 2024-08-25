
const titles = ['Sir', 'Mr', 'Ms', 'Bob', 'Karen'];
const names = ['Crabs', 'Shakealot', 'Greenbacks', 'Yoo'];

const names_used = new Set<string>();

export function generateName(): string {
  const title = titles[Math.floor(Math.random() * titles.length)];
  const name = names[Math.floor(Math.random() * names.length)];
  let nickname = `${title} ${name}`;
  let i = 1;
  while(names_used.has(nickname)) {
    i++;
    nickname = `${title} ${name} ${romanNumeral(i)}`;
  }
  names_used.add(nickname);
  return nickname;
}

const romanLookup: [string, number][] = [
  ['M', 1000],
  ['CM', 900],
  ['D', 500],
  ['CD', 400],
  ['C', 100],
  ['XC', 90],
  ['L', 50],
  ['XL', 40],
  ['X', 10],
  ['IX', 9],
  ['V', 5],
  ['IV', 4],
  ['I', 1],
];

function romanNumeral(i: number): string {
  let roman = '';
  for (const [r, n] of romanLookup) {
    while(i >= n) {
      roman += r;
      i -= n;
    }
  }
  return roman;
}
