const fs = require('fs');
const carriers = require('../docs/carriers.json');

// Helper matcher
function matches(c, patterns) {
  const s = c.slug.toLowerCase();
  const n = c.name.toLowerCase();
  return patterns.some(p => {
    if (typeof p === 'string') {
      return s.includes(p) || n.includes(p);
    } else if (p instanceof RegExp) {
      return p.test(s) || p.test(n);
    }
    return false;
  });
}

const classified = new Map();

function assign(category, filterFn) {
  for (const c of carriers) {
    if (classified.has(c.slug)) continue;
    if (filterFn(c)) {
      classified.set(c.slug, category);
    }
  }
}

// 1. Carga Aérea Global (IATA Air Cargo / Air Waybills)
assign('air_cargo', c => {
  const s = c.slug.toLowerCase();
  const n = c.name.toLowerCase();
  return (
    (s.includes('cargo') && (s.includes('air') || s.includes('aviation') || s.includes('airline'))) ||
    (n.includes('cargo') && (n.includes('air') || n.includes('aviation') || n.includes('airline'))) ||
    s.endsWith('-cargo') || n.endsWith(' cargo') ||
    matches(c, ['skycargo', 'cargolux', 'cargojet', 'air-cargo', 'aviation-cargo', 'airways-cargo', 'calm-air', 'atlas-air'])
  );
});

// 2. Transporte Marítimo Global (Navieras, B/L y Contenedores)
assign('ocean_freight', c => {
  return matches(c, [
    'ocean', 'container', 'line', 'shipping line', 'maritime', 'msc', 'maersk', 'cma-cgm',
    'cosco', 'evergreen', 'hapag', 'one-line', 'yang-ming', 'hmm', 'wan-hai', 'zim', 'oocl',
    'bill of lading', 'dole-ocean', 'culines', 'sinokor', 'matson', 'grimaldi'
  ]);
});

// 3. España
assign('espana', c => {
  return matches(c, [
    'spain', 'españa', '-es', 'seur', 'mrw', 'nacex', 'correosexpress', 'correos-spain',
    'asm-gls-spain', 'paack', 'zeleris', 'celeritas', 'ontime', 'asigna', 'envialia',
    'eco-scooting', 'ecoscooting', 'redur', 'tipsa', 'sending', 'geis', 'servi-entrega-es'
  ]);
});

// 4. Portugal
assign('portugal', c => {
  return matches(c, [
    'portugal', '-pt', 'ctt', 'tourline', 'adicional', 'chronopost-portugal', 'dpd-portugal',
    'delnext', 'via-directa', 'rangel'
  ]);
});

// 5. Alemania / DACH
assign('alemania', c => {
  return matches(c, [
    'germany', 'deutsch', '-de', '-at', '-ch', 'austria', 'swiss', 'switzerland', 'liechtenstein',
    'trans-o-flex', 'hermes-de', 'dpd-germany', 'dpd-austria', 'noerpel', 'amm-spedition',
    'dachser', 'dbschenker', 'hellmann', 'gel-express', 'iloxx', 'dpd-at', 'post-at',
    'austria-mail', 'gebruder-weiss', 'emons', 'spedition', 'dhl-paket'
  ]);
});

// 6. Francia
assign('francia', c => {
  return matches(c, [
    'france', '-fr', 'chronopost-france', 'colis-prive', 'la-poste', 'dpd-france',
    'mondial-relay', 'relais-colis', 'agediss', 'geodis', 'c-chez-vous', 'colissimo',
    'swiship-fr', 'amazon-fr'
  ]);
});

// 7. Reino Unido e Irlanda
assign('uk_irlanda', c => {
  return matches(c, [
    'royal-mail', 'royal mail', 'united kingdom', '-uk', '-ie', 'ireland', 'anpost',
    'evri', 'myhermes-uk', 'yodel', 'dpd-uk', 'dpd-ireland', 'parcelforce', 'whistl',
    'uk-mail', 'apc-overnight', 'arrowxl', 'collect-plus', 'amazon-shipping-uk',
    'swiship-uk', 'dx-delivery', 'tuffnells', 'bjs-home', 'panther'
  ]);
});

// 8. Italia
assign('italia', c => {
  return matches(c, [
    'italy', 'italia', '-it', 'poste-italiane', 'brt', 'bartolini', 'gls-italy',
    'nexive', 'sda', 'arcese', 'arco-spedizioni', 'swiship-it', 'amazon-it', 'fercam', 'cst-italia'
  ]);
});

// 9. Norteamérica (Estados Unidos y Canadá)
assign('norteamerica', c => {
  return matches(c, [
    'usps', 'canada-post', 'united states', 'canada', '-us', '-ca', 'canpar', 'purolator',
    'ontrac', 'lasership', 'spee-dee', 'lone-star', 'cdl', 'day-ross', 'loomis',
    'swiship-com', 'amazon-logistics', 'estes', 'averitt', 'old-dominion', 'r-l-carriers',
    'saia', 'abf-freight', 'pitt-ohio', 'southeastern-freight', 'central-transport',
    'better-trucks', 'intelcom', 'dragonfly-ca', 'uniuni', 'chit-chats'
  ]);
});

// 10. Latinoamérica y Caribe (LATAM)
assign('latam', c => {
  return matches(c, [
    'argentin', 'brazil', 'brasil', 'chile', 'colombia', 'mexico', 'peru', 'uruguay',
    'paraguay', 'bolivia', 'ecuador', 'venezuela', 'panama', 'guatemala', 'cuba',
    'costa-rica', 'el-salvador', 'honduras', 'nicaragua', 'dominican', 'puerto-rico',
    'caribbean', '-br', '-mx', '-ar', '-cl', '-co', '-pe', 'correios', 'chilexpress',
    'estafeta', 'andreani', 'oca', 'servientrega', 'coordinadora', 'interrapidisimo',
    'olva', '99minutos', 'paquetexpress', 'redpack', 'sendex', 'tcc'
  ]);
});

// 11. China continental y Hong Kong
assign('china_hk', c => {
  return matches(c, [
    '-cn', '-hk', 'china', 'hong kong', 'cainiao', '4px', 'yunexpress', 'yanwen', 'sunyou',
    'cne', 'flyt', 'equick', 'chukou1', 'sf-express', 'shentong', 'yto', 'zto', 'best-inc',
    'jd-logistics', 'deppon', 'j&t-china', 'jtexpress-cn', 'bl-cn', 'by56', 'sytrack',
    '139-express', '360lion', 'dhlink', 'tophatter', 'wish'
  ]);
});

// 12. Asia y Oceanía (APAC)
assign('asia_oceania', c => {
  return matches(c, [
    'australia', 'new zealand', 'japan', 'korea', 'singapore', 'malaysia', 'indonesia',
    'thailand', 'philippines', 'vietnam', 'india', 'taiwan', 'pakistan', 'bangladesh',
    '-au', '-nz', '-jp', '-kr', '-sg', '-my', '-id', '-th', '-ph', '-vn', '-in', '-tw',
    'auspost', 'australia-post', 'fastway', 'toll', 'couriers-please', 'delhivery',
    'bluedart', 'ekart', 'shadowfax', 'ninja-van', 'j&t', 'sagawa', 'yamato', 'japan-post',
    'cj-korea', 'korea-post', 'pos-malaysia', 'kerry', 'flash-express', 'jt-express'
  ]);
});

// 13. Europa (Nórdica, Báltica, Central, Este y Balcanes)
assign('resto_europa', c => {
  return matches(c, [
    'poland', 'polska', 'poczta', 'inpost', 'sweden', 'norway', 'denmark', 'finland',
    'estonia', 'latvia', 'lithuania', 'czech', 'slovakia', 'hungary', 'romania', 'bulgaria',
    'greece', 'croatia', 'serbia', 'slovenia', 'albania', 'ukraine', 'belarus', 'russia',
    '-pl', '-se', '-no', '-dk', '-fi', '-ee', '-lv', '-lt', '-cz', '-sk', '-hu', '-ro',
    '-bg', '-gr', '-hr', '-rs', '-si', '-al', '-ua', '-by', '-ru', 'postnord', 'bring',
    'omniva', 'matkahuolto', 'gls-poland', 'dpd-poland', 'fan-courier', 'sameday-ro',
    'speedy-bg', 'econt', 'novaposhta', 'russian-post', 'cdek', 'boxberry'
  ]);
});

// 14. Oriente Medio y África (MENA)
assign('mideast_africa', c => {
  return matches(c, [
    'emirates', 'saudi', 'israel', 'turkey', 'egypt', 'morocco', 'south africa', 'algeria',
    'nigeria', 'kenya', '-ae', '-sa', '-il', '-tr', '-eg', '-ma', '-za', '-ng', '-ke',
    'aramex-sa', 'naqel', 'aymakan', 'saudi-post', 'israel-post', 'ptt', 'yurtici',
    'aras', 'mng', 'postnet', 'ram-za', 'dawn-wing', 'courierit', 'jumia'
  ]);
});

// 15. Global Logistics & Cross-Border (Everything remaining)
assign('global_logistics', c => true);

// Verify total count
const groupCounts = {};
for (const [slug, grp] of classified.entries()) {
  groupCounts[grp] = (groupCounts[grp] || 0) + 1;
}

console.log('Group counts:', groupCounts);
console.log('Total classified:', classified.size, 'out of', carriers.length);
