const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const groups = require('../docs/categorized_carriers.json');

const metadata = {
  espana: {
    title: 'Implementación de Transportistas: España 🇪🇸',
    region: 'España',
    description: 'Servicios de paquetería, mensajería y entrega urgente en territorio español (Correos, SEUR, MRW, Correos Express, Nacex, GLS Spain, etc.).'
  },
  portugal: {
    title: 'Implementación de Transportistas: Portugal 🇵🇹',
    region: 'Portugal',
    description: 'Operadores de correo, express y logística en Portugal (CTT, Tourline/CTT Express, Rangel, Delnext, etc.).'
  },
  alemania: {
    title: 'Implementación de Transportistas: Alemania y Región DACH 🇩🇪 🇦🇹 🇨🇭',
    region: 'Alemania, Austria, Suiza y Liechtenstein',
    description: 'Servicios postales y paquetería en el espacio germano-hablante (Deutsche Post / DHL Paket, DPD DE/AT, Hermes DE, Trans-o-Flex, Swiss Post, Austrian Post, etc.).'
  },
  francia: {
    title: 'Implementación de Transportistas: Francia y Mónaco 🇫🇷 🇲🇨',
    region: 'Francia y Mónaco',
    description: 'Redes de distribución y paquetería en Francia (La Poste, Chronopost, Colis Privé, Mondial Relay, Geodis, Relais Colis, etc.).'
  },
  benelux: {
    title: 'Implementación de Transportistas: Benelux (Países Bajos, Bélgica, Luxemburgo) 🇳🇱 🇧🇪 🇱🇺',
    region: 'Países Bajos, Bélgica y Luxemburgo',
    description: 'Servicios postales y paquetería en Países Bajos, Bélgica y Luxemburgo (PostNL, Bpost, DHL Parcel NL, Dynalogic, Budbee, etc.).'
  },
  uk_irlanda: {
    title: 'Implementación de Transportistas: Reino Unido e Irlanda 🇬🇧 🇮🇪',
    region: 'Reino Unido e Irlanda',
    description: 'Servicios de mensajería y paquetería británica e irlandesa (Royal Mail, Parcelforce, Evri / Hermes UK, Yodel, DPD UK, An Post, etc.).'
  },
  italia: {
    title: 'Implementación de Transportistas: Italia y San Marino 🇮🇹 🇸🇲',
    region: 'Italia, San Marino y Vaticano',
    description: 'Operadores de mensajería y logística en Italia (Poste Italiane, BRT Bartolini, GLS Italy, SDA, Nexive, Arcese, etc.).'
  },
  nordicos_balticos: {
    title: 'Implementación de Transportistas: Países Nórdicos y Bálticos 🇸🇪 🇳🇴 🇩🇰 🇫🇮 🇮🇸 🇪🇪 🇱🇻 🇱🇹',
    region: 'Suecia, Noruega, Dinamarca, Finlandia, Islandia, Estonia, Letonia y Lituania',
    description: 'Redes de entrega en Escandinavia y países bálticos (PostNord, Bring, Posti, Omniva, Matkahuolto, DAO, etc.).'
  },
  europa_este: {
    title: 'Implementación de Transportistas: Europa Central, Oriental y Balcanes 🇵🇱 🇨🇿 🇷🇴 🇬🇷 🇺🇦 🇷🇺',
    region: 'Polonia, República Checa, Eslovaquia, Hungría, Rumanía, Bulgaria, Grecia, Ucrania, Rusia, Balcanes',
    description: 'Operadores postales y empresas de courier en Europa Central, Oriental y los Balcanes (InPost, Poczta Polska, DPD Poland/Romania, Sameday, Fan Courier, Russian Post, CDEK, Nova Poshta, etc.).'
  },
  norteamerica: {
    title: 'Implementación de Transportistas: Norteamérica (Estados Unidos y Canadá) 🇺🇸 🇨🇦',
    region: 'Estados Unidos y Canadá',
    description: 'Servicios postales, paquetería express y carga LTL en Norteamérica (USPS, Canada Post, Purolator, Canpar, OnTrac, LaserShip, Spee-Dee, Estes, etc.).'
  },
  latam: {
    title: 'Implementación de Transportistas: Latinoamérica y Caribe (LATAM) 🇲🇽 🇧🇷 🇦🇷 🇨🇱 🇨🇴 🇵🇪',
    region: 'México, Brasil, Argentina, Chile, Colombia, Perú, Uruguay y resto de LATAM',
    description: 'Redes postales y couriers privados en Latinoamérica (Correios Brasil, Correos de México, Estafeta, 99minutos, Chilexpress, Servientrega, Correo Argentino, Andreani, etc.).'
  },
  china_hk: {
    title: 'Implementación de Transportistas: China continental y Hong Kong 🇨🇳 🇭🇰',
    region: 'China y Hong Kong',
    description: 'Plataformas de paquetería nacional y logística cross-border para eCommerce (Cainiao, China Post, 4PX, YunExpress, Yanwen, SF Express, SunYou, CNE Express, etc.).'
  },
  asia_oceania: {
    title: 'Implementación de Transportistas: Asia y Oceanía (APAC) 🇯🇵 🇰🇷 🇮🇳 🇦🇺 🇳🇿 🇸🇬',
    region: 'Japón, Corea del Sur, India, Australia, Nueva Zelanda, Singapur, Malasia, Indonesia, Tailandia, Filipinas',
    description: 'Servicios postales y couriers en Asia-Pacífico (Australia Post, Japan Post, Yamato, Sagawa, Korea Post, CJ Logistics, Delhivery, Blue Dart, Ninja Van, J&T Express, etc.).'
  },
  mideast_africa: {
    title: 'Implementación de Transportistas: Oriente Medio, Asia Central y África 🇦🇪 🇸🇦 🇮🇱 🇹🇷 🇿🇦 🇪🇬',
    region: 'EAU, Arabia Saudí, Israel, Turquía, Sudáfrica, Egipto, Marruecos y resto de África/MENA',
    description: 'Operadores postales y servicios de entrega en Oriente Medio, Cáucaso, Asia Central y África (Emirates Post, Saudi Post / SPL, Aramex MENA, Naqel, PTT Turquía, Israel Post, South Africa Post, etc.).'
  },
  air_cargo: {
    title: 'Implementación de Transportistas: Carga Aérea Global (IATA Air Cargo / AWB) ✈️',
    region: 'Global (Líneas Aéreas de Carga Internacional)',
    description: 'Seguimiento de guías aéreas Air Waybill (AWB) con soporte de vuelos (Emirates SkyCargo, Qatar Airways Cargo, Lufthansa Cargo, Iberia Cargo, Turkish Cargo, etc.).'
  },
  ocean_freight: {
    title: 'Implementación de Transportistas: Transporte Marítimo y Contenedores (Ocean Freight) 🚢',
    region: 'Global (Navieras y Transporte de Contenedores ISO)',
    description: 'Rastreo de contenedores marítimos y Bill of Lading (B/L) de navieras globales (MSC, Maersk, CMA CGM, COSCO Shipping, Evergreen, Hapag-Lloyd, ONE, HMM, etc.).'
  },
  global_logistics: {
    title: 'Implementación de Transportistas: Redes Globales de Logística y Couriers Cross-Border 🌐',
    region: 'Global (Multinacionales de Mensajería y eCommerce)',
    description: 'Grandes redes logísticas multinacionales, couriers cross-border y servicios de entrega internacional.'
  }
};

function buildIssueBody(key, carrierList) {
  const meta = metadata[key];
  let body = `## 📋 Descripción y Alcance\n\n`;
  body += `${meta.description}\n\n`;
  body += `* **Región / Cobertura**: ${meta.region}\n`;
  body += `* **Total de Transportistas**: ${carrierList.length}\n`;
  body += `* **Integración API**: Endpoint \`POST /trackings\` usando el campo \`shipments[].carrier_hint\` con el slug indicado.\n\n`;

  // Checklist de transportistas
  body += `## 📦 Lista de Servicios a Implementar\n\n`;
  body += `> Marca cada casilla a medida que se implemente y verifique el adaptador correspondiente en \`pkg/carriers/\`.\n\n`;

  for (const c of carrierList) {
    let reqsNote = '';
    if (c.input_requirements) {
      const parts = [];
      if (c.input_requirements.required) parts.push(`Requiere: ${c.input_requirements.required.join(', ')}`);
      if (c.input_requirements.any_of) parts.push(`Cualquiera de: ${c.input_requirements.any_of.join(', ')}`);
      if (c.input_requirements.conditional) parts.push(`Condicional: ${c.input_requirements.conditional.join(', ')}`);
      reqsNote = ` ⚠️ *(${parts.join(' | ')})*`;
    }
    body += `- [ ] **${c.name}** (\`${c.slug}\`)${reqsNote}\n`;
  }

  body += `\n---\n*Generado automáticamente para open-parcels desde el catálogo oficial de Parcels API v4.*`;
  return body;
}

async function createAllIssues() {
  const created = [];
  const entries = Object.entries(groups);

  for (let i = 0; i < entries.length; i++) {
    const [key, carrierList] = entries[i];
    const meta = metadata[key];
    console.log(`[${i + 1}/${entries.length}] Creando issue para: ${meta.title} (${carrierList.length} transportistas)...`);

    const body = buildIssueBody(key, carrierList);
    const tmpFile = path.join(__dirname, `issue_${key}.md`);
    fs.writeFileSync(tmpFile, body);

    try {
      const cmd = `gh issue create --repo gek0v/open-parcels --title "${meta.title.replace(/"/g, '\\"')}" --body-file "${tmpFile}" --label "enhancement,carriers,coverage"`;
      const output = execSync(cmd, { encoding: 'utf8' }).trim();
      console.log(`   ✅ Creado: ${output}`);
      created.push({ key, title: meta.title, url: output, count: carrierList.length });
    } catch (err) {
      console.error(`   ❌ Error creando issue ${key}:`, err.message);
    } finally {
      if (fs.existsSync(tmpFile)) fs.unlinkSync(tmpFile);
    }
  }

  console.log(`\n🎉 Finalizado: ${created.length} issues creados exitosamente.`);
  fs.writeFileSync(path.join(__dirname, '../docs/created_issues.json'), JSON.stringify(created, null, 2));
}

createAllIssues().catch(console.error);
