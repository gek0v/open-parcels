# Catálogo y Matriz de Estado de Transportistas

`open-parcels` cuenta con una arquitectura de dos niveles:
1. **Motor Heurístico de Detección**: Capaz de identificar y validar mediante sumas de comprobación matemática (UPU S10, Modulo 7, Modulo 10, ISO 6346) a **más de 1.500 transportistas y líneas de carga** en 17 catálogos regionales.
2. **Adaptadores de Red Reales (`CarrierAdapter`)**: Controladores dedicados que conectan con APIs REST oficiales, localizadores públicos o servicios B2B empresariales con autenticación.

---

## 🚦 Matriz de Estado de Adaptadores

Para máxima transparencia técnica con la comunidad y usuarios de producción, cada transportista se clasifica según su nivel de integración:

* 🟢 **Live API / Verified**: Conexión de red operativa y verificada contra endpoints públicos o scrapers móviles. No requiere credenciales privadas.
* 🟡 **Enterprise Auth Required**: Adaptador B2B oficial operativo. Requiere configurar variables de entorno con API Key o credenciales OAuth2 (Client Credentials).
* ⚪ **Catalog / Heuristic Only**: Regla de detección regex y algoritmos de checksum registrados en el catálogo. Esperando la contribución de un adaptador de red real por parte de la comunidad.

### Transportistas Principales y Nivel de Integración

| Transportista | Slug | País | Nivel de Integración | Autenticación Requerida | Soporte Handover |
| :--- | :--- | :---: | :---: | :--- | :---: |
| **Cainiao Global** | `cainiao` | 🇨🇳 Global | 🟢 **Live API / Verified** | Ninguna (API pública) | ✅ Sí (Correos Express, CTT) |
| **InPost / Mondial Relay** | `inpost`, `inpost-es` | 🇪🇸 Global | 🟢 **Live API / Verified** | Código postal (`-z`) | — |
| **Correos Express** | `correosexpress` | 🇪🇸 España | 🟢 **Live API / Verified** | Ninguna (REST pública) | ✅ Sí (AliExpress 323) |
| **Correos España** | `correos-spain` | 🇪🇸 España | 🟢 **Live API / Verified** | Ninguna (Localizador web) | ✅ Sí (Oficina / Fecha límite) |
| **SEUR** | `seur` | 🇪🇸 España | 🟢 **Live API / Verified** | Código postal (`-z`) | — |
| **GLS Spain** | `asm-gls-spain` | 🇪🇸 España | 🟢 **Live API / Verified** | Código postal (`-z`) | — |
| **DHL Express** | `dhl-express` | 🌐 Global | 🟡 **Enterprise Auth** | `DHL_API_KEY` | — |
| **DHL Parcel / Paket** | `dhl-parcel` | 🇩🇪 Alemania | 🟡 **Enterprise Auth** | `DHL_API_KEY` | — |
| **UPS** | `ups` | 🇺🇸 Global | 🟡 **Enterprise Auth** | `UPS_CLIENT_ID` + `UPS_CLIENT_SECRET` | — |
| **FedEx** | `fedex` | 🇺🇸 Global | 🟡 **Enterprise Auth** | `FEDEX_CLIENT_ID` + `FEDEX_CLIENT_SECRET` | — |
| **USPS** | `usps` | 🇺🇸 USA | ⚪ *Catalog Only* | Pendiente de adaptador | — |
| **Royal Mail** | `royalmail` | 🇬🇧 UK | ⚪ *Catalog Only* | Pendiente de adaptador | — |
| **Colissimo / La Poste** | `colissimo` | 🇫🇷 Francia | ⚪ *Catalog Only* | Pendiente de adaptador | — |
| **PostNL** | `postnl` | 🇳🇱 Países Bajos | ⚪ *Catalog Only* | Pendiente de adaptador | — |
| **MRW** | `mrw-spain` | 🇪🇸 España | ⚪ *Catalog Only* | Pendiente de adaptador | — |
| **Nacex** | `nacex-spain` | 🇪🇸 España | ⚪ *Catalog Only* | Pendiente de adaptador | — |
| **Paack** | `paack` | 🇪🇸 España | ⚪ *Catalog Only* | Pendiente de adaptador | — |
| **+1.500 Transportistas** | *(ver catálogos)* | 🌍 Global | ⚪ *Catalog Only* | [¡Ayúdanos a implementarlos!](../CONTRIBUTING.md) | — |

---

## 📢 ¡Llamamiento a la Comunidad: Help Us Implement Real Adapters!

El 99% de los transportistas del mercado cuentan con portales de rastreo web accesibles. Si utilizas un operador logístico que actualmente está en nivel ⚪ **Catalog Only**, te invitamos a contribuir con su adaptador real siguiendo la **[Guía de Desarrollo de Adaptadores](CARRIER_GUIDE.md)**.

---

## 1. Cobertura Regional del Catálogo Heurístico

| # | Región / Categoría | Total en Catálogo | Operadores Emblemáticos | Reglas y Checksums |
| :---: | :--- | :---: | :--- | :--- |
| **1** | **China & Hong Kong** 🇨🇳 🇭🇰 | 271 | China Post, SF Express, Hong Kong Post, Yanwen, SunYou, CNE Express, Deppon, YTO, ZTO, 4PX, Cirro | UPU S10 Mod 11, SF 12-14d, YP suffix, SY format, 3A prefix |
| **2** | **Global & Cross-Border** 🌐 | 595 | Aramex, SpeedPAK, Pitney Bowes, Landmark Global, WANBExpress, Winit, CEVA, DSV, Asendia | EE/ES 16d, UPAAB prefix, WNB prefix, AWB format |
| **3** | **Asia & Oceania (APAC)** 🇯🇵 🇰🇷 🇮🇳 🇦🇺 | 125 | Australia Post, Japan Post, Korea Post, India Post, Singapore Post, NZ Post, Ninja Van, Delhivery | UPU S10 Mod 11, Ninja Van prefixes, Delhivery barcodes |
| **4** | **Central/Eastern Europe & Balkans** 🇵🇱 🇨🇿 🇷🇴 | 81 | Poczta Polska, InPost Poland, Czech Post, Romania Post, ELTA Greece, Bulgaria Post, Sameday | UPU S10 Mod 11, InPost 24-digit, Sameday barcodes |
| **5** | **North America** 🇺🇸 🇨🇦 | 42 | USPS, UPS, FedEx, Canada Post, Purolator, Canpar, OnTrac, Intelcom Express, Chit Chats | UPU S10, USPS 20-22d, UPS 1Z Mod 10, Canada 16d, Purolator PIN |
| **6** | **Nordic & Baltic Countries** 🇸🇪 🇳🇴 🇩🇰 🇫🇮 | 23 | PostNord Sweden, Posten Norge, Posti Finland, Omniva Estonia, Latvia Post, Lithuania Post, Bring | UPU S10 Mod 11, Posti JJFI, Bring 70-prefix, DAO barcodes |
| **7** | **Spain** 🇪🇸 | 24 | Correos España, Correos Express, CTT Express, SEUR, Paack, MRW, GLS Spain, TIPSA, Zeleris | UPU S10, 16/23d domestic, SEUR 14d, MRW franchise codes |
| **8** | **Global Air Cargo (IATA AWB)** ✈️ | 136 | Emirates SkyCargo, Qatar Airways Cargo, Lufthansa Cargo, Iberia Cargo, Turkish Cargo | IATA AWB `NNN-NNNNNNNN` with Mod 7 checksum |
| **9** | **Italy & San Marino** 🇮🇹 🇸🇲 | 19 | Poste Italiane, SDA Express, BRT Bartolini, GLS Italy, Nexive | UPU S10 Mod 11, SDA domestic, BRT 12-digit barcodes |
| **10** | **France & Monaco** 🇫🇷 🇲🇨 | 22 | Colissimo, La Poste, Chronopost France, Colis Privé, Relais Colis, Geodis, Mondial Relay | UPU S10, 6A/6M domestic, Chronopost barcodes |
| **11** | **United Kingdom & Ireland** 🇬🇧 🇮🇪 | 38 | Royal Mail, Parcelforce, Evri (Hermes UK), Yodel, DPD UK, An Post Ireland, Whistl | UPU S10 Mod 11, Evri 16-digit, Yodel JD prefix |
| **12** | **Middle East & Africa (MENA)** 🇦🇪 🇸🇦 🇮🇱 | 57 | Emirates Post, Saudi Post (SPL), Turkey Post (PTT), Israel Post, South Africa Post, Naqel | UPU S10 Mod 11, Naqel NQ-prefix, PTT domestic |
| **13** | **Germany & DACH Region** 🇩🇪 🇦🇹 🇨🇭 | 26 | Deutsche Post, DHL Paket, Swiss Post, Austrian Post, Hermes Germany, Trans-o-Flex | UPU S10, Swiss 18d, Deutsche Post 0034043 |
| **14** | **Latin America & Caribbean (LATAM)** 🇲🇽 🇧🇷 🇦🇷 | 25 | Correios Brazil, Correos de México, Correo Argentino, Serpost Peru, Correo Uruguayo, Estafeta | UPU S10 Mod 11, Estafeta 22-digit waybills |
| **15** | **Ocean Freight & Shipping Containers** 🚢 | 49 | MSC, Maersk Line, CMA CGM, Evergreen, COSCO Shipping, Hapag-Lloyd, ONE, ZIM | ISO 6346 4-letter + 7-digit container standard |
| **16** | **Benelux** 🇳🇱 🇧🇪 🇱🇺 | 18 | PostNL, Bpost Belgium, Luxembourg Post, DHL Parcel Netherlands, Budbee | UPU S10 Mod 11, PostNL 3S barcodes, Bpost 24d |
| **17** | **Portugal** 🇵🇹 | 12 | CTT Portugal, CTT Expresso, Chronopost Portugal, DPD Portugal, Delnext | UPU S10 Mod 11, CTT domestic |

---

## 2. Algoritmos de Suma de Comprobación Soportados

1. **UPU S10 Modulo 11**: Ponderaciones `[8, 6, 4, 2, 3, 5, 9, 7]` contra dígito de control.
2. **UPS 1Z Modulo 10**: Ponderaciones alternas (1, 2) alfanuméricas sobre caracteres 2..16.
3. **IATA Air Cargo Modulo 7**: Comprobación directa `serial % 7 == check`.
4. **ISO 6346 Contenedores Marítimos**: 4 letras de propietario/categoría + 6 dígitos serie + 1 dígito control con pesos potencia de 2.
