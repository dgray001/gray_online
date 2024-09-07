import {getRandom} from "../../../../scripts/util";

type NameGender = 'male' | 'female' | 'androgynous';

type NameCombination = [boolean, boolean, boolean, boolean];

const prefixes: Map<NameGender, string[]> = new Map([
  ['male', ['Sir', 'Mr.']],
  ['female', ['Mrs.', 'Ms.', 'Maam', 'Mistress', 'Madam']],
  ['androgynous', ['Professor', 'Governor', 'Master', 'Dr.']],
]);

const prefixes_rare: Map<NameGender, string[]> = new Map([
  ['male', ['Brother', 'Caliph', 'Chairman', 'Count', 'Earl', 'Goodman', 'His Eminence', 'Imam', 'Lord', 'Patriarch', 'Shah']],
  ['female', ['Sister', 'Countess', 'Chairwoman', 'Goodwife', 'Lady', 'Shahbanu']],
  ['androgynous', ['Admiral', 'General', 'Consul', 'Elder', 'Emeritus', 'Goody', 'Mayor', 'President', 'Princeps', 'Doctor']],
]);

const first_names: Map<NameGender, string[]> = new Map([
  ['male', [
    'Nicolás', 'Mateo', 'Juan', 'Benicio', 'Daniel', 'Kevin', 'Luis', 'Théo', 'Miguel',
    'Noah', 'Lucas', 'Paulo', 'Liam', 'Benjamin', 'Jack', 'Thomas', 'Nathan', 'Gaspar',
    'Maximiliano', 'Santiago', 'Thiago', 'Inuk', 'Malik', 'Qillaq', 'Stevenson', 'Ali',
    'Wilson', 'David', 'Ricardo', 'Aiden', 'Josiah', 'Santiago', 'Matías', 'Thiago',
    'Dylan', 'Ramón', 'César', 'Gael', 'Ian', 'Oliver', 'Elijah', 'Theodore', 'Henry',
    'James', 'John', 'Robert', 'Michael', 'William', 'Richard', 'Charles', 'Joseph',
    'Valentín', 'Christopher', 'Matthew', 'Anthony', 'Mark', 'Donald', 'Kenneth', 'Mohamed',
    'Noel', 'Iker', 'Paul', 'Mateo', 'Matsvey', 'Arthur', 'Davud', 'Aleksandar',
    'Luka', 'Andreas', 'Jakub', 'Noah', 'Hugo', 'Bartal', 'Eino', 'Andria', 'Finn',
    'Kai', 'Georgios', 'Freddie', 'Dominik', 'Leonardo', 'Elias', 'Aleksandar', 'Levi',
    'Cillian', 'Nikodem', 'Andrei', 'Mikhail', 'Dmitriy', 'Sergei', 'Tommaso', 'Ivan',
    'Harris', 'Rory', 'Dragan', 'Artem', 'Archie', 'Vladyslav', 'Pablo', 'Markuss'
  ]],
  ['female', [
    'Sofía', 'Alysha', 'Martha', 'Helena', 'Maria', 'Olivia', 'Alice', 'Emma',
    'Renata', 'Emily', 'Roxana', 'Isis', 'Aline', 'Charlotte', 'Amelia', 'Sophia',
    'Chloe', 'Léa', 'Guadalupe', 'Mía', 'Aitana', 'Ivaana', 'Niviaq', 'Pipaluk',
    'Widelene', 'Antonella', 'Esther', 'Amelia', 'Kyra', 'Sofía', 'Camila', 'Valentina',
    'Aurora', 'Luna', 'Samantha', 'Camila', 'Julieta', 'Evelyn', 'Mary', 'Linda',
    'Barbara', 'Elizabeth', 'Jennifer', 'Susan', 'Dorothy', 'Margaret', 'Patricia',
    'Jessica', 'Karen', 'Sarah', 'Lisa', 'Nancy', 'Sandra', 'Betty', 'Ashley', 'Kimberly',
    'Amelia', 'Ambra', 'Martina', 'Hannah', 'Sophia', 'Ksenia', 'Juliette', 'Iman',
    'Viktoria', 'Gabriela', 'Nika', 'Eleni', 'Panagiota', 'Eliška', 'Frida', 'Agnes',
    'Isla', 'Sienna', 'Saara', 'Sólja', 'Björg', 'Aino', 'Jade', 'Ambre', 'Mariami',
    'Chloe', 'Eleni', 'Angeliki', 'Daisy', 'Zoé', 'Katla', 'Emīlija', 'Frieda', 'Marija',
    'Jana', 'Anastasia', 'Sofija', 'Oliwia', 'Zofia', 'Aurora', 'Milica', 'Elsa', 'Ona'
  ]],
  ['androgynous', ['Taylor', 'Logan', 'Riley', 'Parker', 'Avery', 'Jordan', 'Casey', 'Azariah', 'August']],
]);

const first_names_rare: Map<NameGender, string[]> = new Map([
  ['male', [
    'Omar', 'Mustafa', 'Youssef', 'Hamza', 'Hussein', 'Ibrahim', 'Ali', 'Lwandle',
    'Halim', 'Peter', 'Pierre', 'Mina', 'Habib', 'Fadi', 'Djamel', 'Amir','Mehdi',
    'Karim', 'Tareq', 'Mamadou', 'Aziz', 'José', 'Omphile', 'Ofentse', 'Amine',
    'Abdullah', 'Fahd', 'Davit', 'Tigran', 'Armen', 'Samvel', 'Raul', 'Vugar', 'Mùchén',
    'Hàoyǔ', 'Mùchén', 'Yǔzé', 'Mùyáng', 'Mohammed', 'Muhammad', 'Abdul', 'Vedant',
    'Sutrisno', 'Herman', 'Amir-Ali', 'Samyar', 'Reza', 'Abbas', 'Hydar', 'Adam',
    'Lavi', 'Moshe', 'Charbel', 'Aoi', 'Sō', 'Nagi', 'Ren', 'Tatsuki', 'Haruma',
    'Alikhan', 'Aldiyar', 'Abdulaziz', 'Elie', 'Christian', 'Joe', 'Wan', 'Izz',
    'Tenun', 'Tushig', 'Krishna', 'Bishal', 'Prem', 'Bilal', 'Ezekiel', 'Turki',
    'Seo-jun', 'Ji-ho', 'Min-jun', 'Chun-hung', 'Chia-hao', 'Yusuf', 'Somchai',
    'Somporn', 'Alparslan', 'Miraç', 'Göktug', 'Mehmet', 'Hüseyin', 'İsmail', 'Imronbek',
    'Leo', 'Manea', 'Kayden', 'Luca', 'Ariki', 'Hiro', 'Teiva', 'Tapuarii', 'Kiwa',
  ]],
  ['female', [
    'Imene', 'Yasmine', 'Maria', 'Meriem', 'Melissa', 'Lydia', 'Fatma', 'Hosniya',
    'Fatin', 'Aya', 'Suha', 'Habiba', 'Marie', 'Marwa', 'Nada', 'Salma', 'Isabel',
    'Mona', 'Assia', 'Inès', 'Doha', 'Irene', 'Shayma', 'Khawla', 'Nora', 'Amira',
    'Maryam', 'Lyn', 'Hur', 'Lucy', 'Nare', 'Arpi', 'Eva', 'Anna', 'Anahit', 'Hasmik',
    'Zahra', 'Aylin', 'Zeynab', 'İnci', 'Nuray', 'Sevinj', 'Gunay', 'Ruòxī', 'Yìhán',
    'Yīnuò', 'Yǐmò', 'Yǔtóng', 'Yǔxī', 'Aditi', 'Aarya', 'Vamika', 'Mehar', 'Nurhayati',
    'Sunarti', 'Fatemeh', 'Mersana', 'Zahra', 'Zeinab', 'Avigail', 'Yael', 'Adel',
    'Tamar', 'Libi', 'Shams', 'Jori', 'Nur', 'Celine', 'Yasmin', 'Himari', 'Hinata',
    'Hina', 'Uta', 'Sakura', 'Jouri', 'Leen', 'Wateen', 'Mädïna', 'Ayşa', 'Tomyris',
    'Hussa', 'Saliha', 'Marie', 'Amin-Erdene', 'Michele', 'Sondor', 'Dhia', 'Shristi',
    'Binita', 'Fozia', 'Ji-an', 'Ha-rin', 'Shu-fen', 'Sumayah', 'Zeynep', 'Soliha',
    'Isla', 'Matilda', 'Kiana', 'Ohana', 'Kaia', 'Zoé', 'Willow', 'Marama', 'Aroha'
  ]],
  ['androgynous', ['Rowan', 'Malak', 'Karen']],
]);

const last_names = [
  'Crabs', 'Shakealot', 'Greenbacks', 'Yoo', 'Arco', 'Astley', 'Baldwins', 'Bourbon',
  'Burgundy', 'Capet', 'Carolingian', 'Collalto', 'Estridsen', 'Ernušt', 'Ferdinand',
  'Gravenreuth', 'Gusic', 'Habsburg', 'Hesse', 'Holstein', 'Kurki', 'Lombards',
  'Lothbrok', 'Madi', 'Medici', 'Morosini', 'Nassau', 'Orange', 'Orsini', 'Pfeffel',
  'Priuli', 'Querini', 'Rothschild', 'Sanudo', 'Schleswig', 'Sparneck', 'Strozzi',
  'Tudor', 'von Schmitt', 'Windsor', 'Smith', 'Johnson', 'Williams', 'García',
  'Davis', 'Martínez', 'González', 'Anderson', 'Moore', 'Pérez', 'White', 'Clark',
  'Robinson', 'Nguyen', 'Campbell', 'Nelson', 'Green', 'Parker', 'Turner', 'Evans',
  'Wang', 'Mahamat', 'Mohamed', 'Ali', 'Ahmed', 'Ibrahim', 'Hassan', 'Li', 'Zhang',
  'Chen', 'Liu', 'Lu', 'Zhou', 'Müller', 'Hernández', 'da Silva', 'Ferreira', 'Peter',
  'Madden', 'Whitlock', 'Langston', 'Elrod', 'Yarbrough', 'Singh', 'Fleet', 'Ninomae',
  'Darcy', 'Albertine', 'Sapphirus', 'Maverick', 'Boney', 'Rollapull', 'Whackamattus'
];

const suffixes: Map<NameGender, string[]> = new Map([
  ['androgynous', ['Esq.', 'Senior', 'Junior', 'Sr.', 'Jr.', 'II', 'III']],
]);

const suffixes_rare: Map<NameGender, string[]> = new Map([
  ['androgynous', ['the Elder', 'the Younger', 'JD', 'MA', 'MD', 'PhD', 'the Wise']],
]);

const name_combinations: NameCombination[] = [
  [true, true, true, true],
  [false, true, true, true],
  [true, false, true, true],
  [true, true, false, true],
  [true, true, true, false], [true, true, true, false],
  [false, false, true, true],
  [false, true, false, true],
  [false, true, true, false], [false, true, true, false], [false, true, true, false],
    [false, true, true, false], [false, true, true, false], [false, true, true, false],
  [true, false, true, false], [true, false, true, false], [true, false, true, false],
  [true, true, false, false], [true, true, false, false], [true, true, false, false],
  [false, true, false, false], [false, true, false, false], [false, true, false, false],
    [false, true, false, false], [false, true, false, false], [false, true, false, false],
  [false, false, true, false], [false, false, true, false], [false, false, true, false],
];

function randomizeGender(): NameGender {
  const r = Math.random();
  if (r < 0.20) {
    return 'androgynous';
  } else if (r < 0.6) {
    return 'female';
  } else {
    return 'male';
  }
}

function getName(names_common: Map<NameGender, string[]>, names_rare: Map<NameGender, string[]>, gender: NameGender): string {
  let m = names_common;
  if (Math.random() < 0.25 && !!names_rare) {
    m = names_rare;
  }
  const possible_names: string[] = [];
  possible_names.push(...(m.get('androgynous') ?? []));
  if (gender === 'male' || gender === 'female') {
    possible_names.push(...(m.get(gender) ?? []));
  }
  return getRandom(possible_names);
}

const names_used = new Set<string>();

export function generateName(): string {
  const name_combo = getRandom(name_combinations);
  const gender = randomizeGender();
  let nickname = '';
  if (name_combo[0]) {
    nickname = getName(prefixes, prefixes_rare, gender);
  }
  if (name_combo[1]) {
    nickname += ` ${getName(first_names, first_names_rare, gender)}`;
  }
  if (name_combo[2]) {
    nickname += ` ${getRandom(last_names)}`;
  }
  if (name_combo[3]) {
    nickname += `, ${getName(suffixes, suffixes_rare, gender)}`;
  }
  let i = 1;
  while(names_used.has(nickname)) {
    i++;
    nickname += ` ${romanNumeral(i)}`;
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
