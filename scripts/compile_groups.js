const fs = require('fs');
const path = require('path');
const carriers = require('../docs/carriers.json');

const assigned = new Map();

function hasAssigned(slug) {
  return assigned.has(slug);
}

function assign(slug, group) {
  if (!assigned.has(slug)) {
    assigned.set(slug, group);
  }
}

// 1. Carga Aérea Global (IATA Air Cargo / Air Waybills)
for (const c of carriers) {
  if (hasAssigned(c.slug)) continue;
  const s = c.slug.toLowerCase();
  const n = c.name.toLowerCase();
  if (
    (s.includes('cargo') && (s.includes('air') || s.includes('aviation') || s.includes('airline'))) ||
    (n.includes('cargo') && (n.includes('air') || n.includes('aviation') || n.includes('airline'))) ||
    s.endsWith('-cargo') || n.endsWith(' cargo') ||
    s.includes('skycargo') || s.includes('cargolux') || s.includes('cargojet') ||
    n.includes('skycargo') || n.includes('cargolux') || n.includes('cargojet') ||
    s.includes('aviation') || n.includes('aviation') ||
    s.includes('airlines') || n.includes('airlines')
  ) {
    assign(c.slug, 'air_cargo');
  }
}

// 2. Transporte Marítimo Global (Navieras, B/L y Contenedores)
const oceanKeywords = [
  'ocean', 'container', 'shipping line', 'line', 'maritime', 'msc', 'maersk', 'cma-cgm',
  'cosco', 'evergreen', 'hapag', 'one-line', 'yang-ming', 'hmm', 'wan-hai', 'zim', 'oocl',
  'bill of lading', 'dole-ocean', 'culines', 'sinokor', 'matson', 'grimaldi', 'containerships',
  'sealand', 'nyk', 'mol', 'k-line', 'hamburg sud', 'safmarine', 'boluda', 'crowley'
];
for (const c of carriers) {
  if (hasAssigned(c.slug)) continue;
  const s = c.slug.toLowerCase();
  const n = c.name.toLowerCase();
  if (oceanKeywords.some(k => s.includes(k) || n.includes(k))) {
    assign(c.slug, 'ocean_freight');
  }
}

// 3. Suffix ISO code mapping
const suffixMap = {
  // España
  'es': 'espana',
  // Portugal
  'pt': 'portugal',
  // Alemania & DACH
  'de': 'alemania', 'at': 'alemania', 'ch': 'alemania', 'li': 'alemania',
  // Francia
  'fr': 'francia', 'mc': 'francia',
  // Benelux
  'nl': 'benelux', 'be': 'benelux', 'lu': 'benelux',
  // UK & Irlanda
  'uk': 'uk_irlanda', 'gb': 'uk_irlanda', 'ie': 'uk_irlanda',
  // Italia
  'it': 'italia', 'sm': 'italia', 'va': 'italia',
  // Nórdicos & Bálticos
  'se': 'nordicos_balticos', 'no': 'nordicos_balticos', 'dk': 'nordicos_balticos',
  'fi': 'nordicos_balticos', 'is': 'nordicos_balticos', 'ee': 'nordicos_balticos',
  'lv': 'nordicos_balticos', 'lt': 'nordicos_balticos',
  // Europa Central, Oriental y Balcanes
  'pl': 'europa_este', 'cz': 'europa_este', 'sk': 'europa_este', 'hu': 'europa_este',
  'ro': 'europa_este', 'bg': 'europa_este', 'ua': 'europa_este', 'by': 'europa_este',
  'md': 'europa_este', 'ru': 'europa_este', 'gr': 'europa_este', 'cy': 'europa_este',
  'hr': 'europa_este', 'rs': 'europa_este', 'si': 'europa_este', 'ba': 'europa_este',
  'al': 'europa_este', 'mk': 'europa_este', 'me': 'europa_este', 'xk': 'europa_este',
  // Norteamérica
  'us': 'norteamerica', 'ca': 'norteamerica',
  // LATAM
  'mx': 'latam', 'br': 'latam', 'ar': 'latam', 'cl': 'latam', 'co': 'latam',
  'pe': 'latam', 'uy': 'latam', 'py': 'latam', 'ec': 'latam', 'bo': 'latam',
  've': 'latam', 'pa': 'latam', 'cr': 'latam', 'gt': 'latam', 'sv': 'latam',
  'hn': 'latam', 'ni': 'latam', 'do': 'latam', 'cu': 'latam', 'pr': 'latam',
  'tt': 'latam', 'jm': 'latam', 'aw': 'latam', 'cw': 'latam',
  // China & HK
  'cn': 'china_hk', 'hk': 'china_hk', 'mo': 'china_hk',
  // Asia & Oceanía
  'jp': 'asia_oceania', 'kr': 'asia_oceania', 'tw': 'asia_oceania', 'in': 'asia_oceania',
  'au': 'asia_oceania', 'nz': 'asia_oceania', 'sg': 'asia_oceania', 'my': 'asia_oceania',
  'id': 'asia_oceania', 'th': 'asia_oceania', 'ph': 'asia_oceania', 'vn': 'asia_oceania',
  'pk': 'asia_oceania', 'bd': 'asia_oceania', 'lk': 'asia_oceania', 'np': 'asia_oceania',
  'kh': 'asia_oceania', 'la': 'asia_oceania', 'mm': 'asia_oceania', 'fj': 'asia_oceania',
  // Oriente Medio & África
  'ae': 'mideast_africa', 'sa': 'mideast_africa', 'il': 'mideast_africa', 'tr': 'mideast_africa',
  'za': 'mideast_africa', 'eg': 'mideast_africa', 'ma': 'mideast_africa', 'ng': 'mideast_africa',
  'ke': 'mideast_africa', 'qa': 'mideast_africa', 'kw': 'mideast_africa', 'kz': 'mideast_africa',
  'uz': 'mideast_africa', 'ge': 'mideast_africa', 'am': 'mideast_africa', 'az': 'mideast_africa',
  'om': 'mideast_africa', 'bh': 'mideast_africa', 'jo': 'mideast_africa', 'lb': 'mideast_africa',
  'gh': 'mideast_africa', 'tz': 'mideast_africa', 'ug': 'mideast_africa', 'et': 'mideast_africa',
  'dz': 'mideast_africa', 'tn': 'mideast_africa', 'mu': 'mideast_africa', 'sn': 'mideast_africa',
  'ci': 'mideast_africa', 'cm': 'mideast_africa', 'rw': 'mideast_africa', 'bw': 'mideast_africa',
  'na': 'mideast_africa', 'zw': 'mideast_africa', 'mz': 'mideast_africa', 'ao': 'mideast_africa'
};

for (const c of carriers) {
  if (hasAssigned(c.slug)) continue;
  const parts = c.slug.toLowerCase().split('-');
  const last = parts[parts.length - 1];
  if (suffixMap[last]) {
    assign(c.slug, suffixMap[last]);
  }
}

// 4. Detailed regional & country keywords:
const countryRules = [
  // España
  {
    group: 'espana',
    terms: ['spain', 'españa', 'seur', 'mrw', 'nacex', 'correosexpress', 'correos-spain', 'spain post', 'asm-gls', 'paack', 'zeleris', 'celeritas', 'ontime', 'asigna', 'envialia', 'ecoscooting', 'redur', 'tipsa', 'sending', 'servi-entrega-es', 'activos24', 'packlink', 'tourline']
  },
  // Portugal
  {
    group: 'portugal',
    terms: ['portugal', 'ctt', 'rangel', 'delnext', 'via directa', 'adicional logistics', 'portugal post']
  },
  // Alemania & DACH
  {
    group: 'alemania',
    terms: ['germany', 'deutsch', 'austria', 'swiss', 'switzerland', 'liechtenstein', 'trans-o-flex', 'hermes de', 'dpd germany', 'dpd austria', 'noerpel', 'dachser', 'db schenker', 'hellmann', 'gel express', 'iloxx', 'austria post', 'austrian post', 'gebrüder weiss', 'gebruder', 'emons', 'spedition', 'dhl paket', 'postat', 'swiss post', 'swisspost', 'amm spedition', 'cepra', 'day one']
  },
  // Francia
  {
    group: 'francia',
    terms: ['france', 'chronopost', 'colis prive', 'colis privé', 'la poste', 'dpd france', 'mondial relay', 'relais colis', 'agediss', 'geodis', 'c chez vous', 'colissimo', 'swiship fr', 'amazon shipping france', 'french post', 'ciblex']
  },
  // Benelux (Países Bajos, Bélgica, Luxemburgo)
  {
    group: 'benelux',
    terms: ['netherlands', 'belgium', 'luxembourg', 'postnl', 'bpost', 'dynalogic', 'cycloon', 'dhl parcel netherlands', 'swiship nl', 'budbee', 'bringly', 'belgian post', 'dutch post', 'postlux', 'dragonfly netherlands']
  },
  // UK & Irlanda
  {
    group: 'uk_irlanda',
    terms: ['royal mail', 'united kingdom', 'ireland', 'an post', 'irish post', 'evri', 'myhermes', 'yodel', 'dpd uk', 'dpd ireland', 'parcelforce', 'whistl', 'apc overnight', 'arrowxl', 'collect+', 'amazon shipping uk', 'swiship uk', 'dx delivery', 'tuffnells', 'bjs home delivery', 'panther logistics', 'ao uk', 'ait home delivery uk', 'drop2shop']
  },
  // Italia
  {
    group: 'italia',
    terms: ['italy', 'italia', 'poste italiane', 'brt bartolini', 'gls italy', 'nexive', 'sda express', 'arcese', 'arco spedizioni', 'swiship it', 'amazon italy', 'fercam', 'cst italia', 'sailpost', 'italian post', 'alfred 24']
  },
  // Nórdicos & Bálticos
  {
    group: 'nordicos_balticos',
    terms: ['sweden', 'norway', 'denmark', 'finland', 'iceland', 'estonia', 'latvia', 'lithuania', 'postnord', 'bring', 'omniva', 'matkahuolto', 'posti', 'dropp', 'earlybird', 'city mail', 'helthjem', 'danske fragt', 'dao', 'collicare', 'sweden post', 'norway post', 'finland post', 'denmark post', 'iceland post', 'estonia post', 'latvia post', 'lithuania post', 'cargobus', 'coolrunner']
  },
  // Europa Central, Oriental y Balcanes
  {
    group: 'europa_este',
    terms: ['poland', 'polska', 'poczta', 'inpost', 'czech', 'slovakia', 'hungary', 'romania', 'bulgaria', 'greece', 'croatia', 'serbia', 'slovenia', 'albania', 'ukraine', 'belarus', 'russia', 'gls poland', 'dpd poland', 'fan courier', 'sameday', 'speedy', 'econt', 'novaposhta', 'russian post', 'cdek', 'boxberry', 'acs courier', 'boxnow', 'dpexpress', 'poland post', 'czech post', 'slovakia post', 'hungary post', 'romania post', 'bulgaria post', 'greece post', 'croatia post', 'serbia post', 'slovenia post', 'albania post', 'ukraine post', 'belpost', 'dellin', 'baikal service', 'bex express', 'csomagpiac', 'easymail', 'elta courier', 'expressone', 'famafutar', 'depo']
  },
  // Norteamérica (Estados Unidos y Canadá)
  {
    group: 'norteamerica',
    terms: ['united states', 'canada', 'usps', 'canada post', 'purolator', 'canpar', 'ontrac', 'lasership', 'spee-dee', 'lone star', 'cdl', 'day & ross', 'loomis', 'swiship us', 'estes express', 'averitt', 'old dominion', 'r+l carriers', 'saia', 'abf freight', 'pitt ohio', 'better trucks', 'intelcom', 'dragonfly canada', 'uniuni', 'chit chats', 'aaa cooper', 'a duie pyle', 'daylight transport', 'dayton freight', 'dohrn', 'central transport', 'crosscountry', 'deliver it', 'e24', 'apple express', 'capital express', 'coyote logistics', 'domain logistics']
  },
  // LATAM & Caribe
  {
    group: 'latam',
    terms: ['argentin', 'brazil', 'brasil', 'chile', 'colombia', 'mexico', 'méxico', 'peru', 'perú', 'uruguay', 'paraguay', 'bolivia', 'ecuador', 'venezuela', 'panama', 'panamá', 'guatemala', 'cuba', 'costa rica', 'el salvador', 'honduras', 'nicaragua', 'dominican', 'puerto rico', 'caribbean', 'correios', 'chilexpress', 'estafeta', 'andreani', 'oca', 'servientrega', 'coordinadora', 'interrapidisimo', 'olva', '99minutos', 'paquetexpress', 'redpack', 'sendex', 'tcc', 'brazil post', 'argentine post', 'chile post', 'colombia post', 'mexico post', 'peru post', 'uruguay post', 'correo argentino', 'correos cuba', 'correos de guatemala', 'correos el salvador', 'correos panama', 'cpost international', 'caribou', 'bluexpress', 'starken']
  },
  // China & HK
  {
    group: 'china_hk',
    terms: ['china', 'hong kong', 'cainiao', '4px', 'yunexpress', 'yanwen', 'sunyou', 'sytrack', 'cne express', 'flyt', 'equick', 'chukou1', 'sf express', 'shentong', 'yto', 'zto', 'best inc', 'jd logistics', 'deppon', 'j&t china', 'dhlink', 'tophatter', 'wish', 'anbeiqi', 'beishangtong', 'blue leopard', 'bluebull', 'bulong', 'by56', 'cititrans', 'ddgyl', 'edatong', 'ezex', 'fafalux', 'fanglian', 'china post', 'hong kong post', '139 express', '360lion', '3pe express', '70mai', 'alljoy', 'anserx', 'aochen', 'aoyue', 'asgyl', 'asiafly', 'asus international', 'bab kingtrans', 'beone', 'bestfulfill', 'bird system', 'blkj', 'boyuan', 'bsb', 'btd', 'buffalo', 'by56', 'byt', 'cacesa', 'cce', 'cel', 'cirro', 'cn express', 'cred', 'cross border runteck', 'ctty', 'cxc express', 'datt express', 'dd express', 'deqi', 'dghy', 'djx', 'dm fulfillment', 'dongdi', 'dpdex', 'duxiuexp', 'dwe', 'dye express', 'dyj', 'dztgj', 'e d post', 'eac', 'ec-firstclass', 'ecsc', 'efs asia', 'equick china', 'ese', 'etdropship', 'ets express', 'ewe', 'ews profit', 'ezex', 'fan sheng', 'far good']
  },
  // Asia & Oceanía
  {
    group: 'asia_oceania',
    terms: ['australia', 'new zealand', 'japan', 'korea', 'singapore', 'malaysia', 'indonesia', 'thailand', 'philippines', 'vietnam', 'india', 'taiwan', 'pakistan', 'bangladesh', 'auspost', 'australia post', 'fastway', 'toll', 'couriers please', 'delhivery', 'bluedart', 'ekart', 'shadowfax', 'ninja van', 'j&t', 'sagawa', 'yamato', 'japan post', 'cj korea', 'korea post', 'pos malaysia', 'kerry express', 'flash express', 'jt express', '2go', 'airroad', 'airpak express', 'allied express', 'anteraja', 'bigpost', 'border express', 'call courier', 'cambodia post', 'castle parcels', 'cebu pacific', 'chita', 'chunil cargo', 'citilogistics', 'dakota cargo', 'delivere', 'deliverybox', 'dtdc', 'echindia', 'ecohai', 'fast courier australia', 'gati', 'hunter express', 'indonesia post', 'pakistan post', 'phlpost', 'singpost', 'thailand post', 'vietnam post']
  },
  // Oriente Medio & África
  {
    group: 'mideast_africa',
    terms: ['emirates', 'saudi', 'israel', 'turkey', 'egypt', 'morocco', 'south africa', 'algeria', 'nigeria', 'kenya', 'naqel', 'aymakan', 'saudi post', 'israel post', 'ptt', 'yurtici', 'aras kargo', 'mng kargo', 'postnet', 'ram tracking', 'dawn wing', 'courierit', 'jumia', 'kazpost', 'ajex', 'akis express', 'albanian courier', 'albania post', 'armenia post', 'aruba post', 'ase lojistik', 'azerbaijan post', 'bahrain post', 'botswana post', 'brunei post', 'burd delivery', 'bxb delivery', 'c3x', 'cheetah delivery', 'cyprus post', 'dexpress', 'dpd south africa', 'eco freight', 'egypt post', 'elite express cargo', 'els', 'emex', 'emirates post', 'ethiopia post', 'fareye', 'fastco', 'fetchr', 'first logistics', 'georgia post', 'ghana post', 'iran post', 'iraq post', 'jordan post', 'kuwait post', 'lebanon post', 'mauritius post', 'namibia post', 'nigeria post', 'oman post', 'qatar post', 'rwanda post', 'senegal post', 'tanzania post', 'tunisia post', 'uganda post', 'zambia post', 'zimbabwe post']
  }
];

for (const rule of countryRules) {
  for (const c of carriers) {
    if (hasAssigned(c.slug)) continue;
    const s = c.slug.toLowerCase();
    const n = c.name.toLowerCase();
    if (rule.terms.some(t => s.includes(t) || n.includes(t))) {
      assign(c.slug, rule.group);
    }
  }
}

// 5. Fallback: global_logistics
for (const c of carriers) {
  if (!hasAssigned(c.slug)) {
    assign(c.slug, 'global_logistics');
  }
}

// Generate summary
const groupsMap = {};
for (const c of carriers) {
  const grp = assigned.get(c.slug);
  if (!groupsMap[grp]) groupsMap[grp] = [];
  groupsMap[grp].push(c);
}

console.log('--- REPORTE DETALLADO POR GRUPO ---');
let total = 0;
for (const [k, v] of Object.entries(groupsMap).sort((a,b) => b[1].length - a[1].length)) {
  console.log(`${k}: ${v.length} transportistas`);
  total += v.length;
}
console.log('TOTAL ASIGNADOS:', total, 'de', carriers.length);

fs.writeFileSync(path.join(__dirname, '../docs/categorized_carriers.json'), JSON.stringify(groupsMap, null, 2));
