# Supported Carriers & Regional Coverage

`open-parcels` supports **more than 1,500 logistics carriers, postal operators, and freight forwarders** worldwide, organized across 17 regional and functional catalogs.

---

## 1. Regional Breakdown

| # | Region / Category | Carriers Count | Key Flagship Operators | Pattern / Checksum Support |
| :---: | :--- | :---: | :--- | :--- |
| **1** | **China & Hong Kong** 🇨🇳 🇭🇰 | 271 | China Post, SF Express, Hong Kong Post, Yanwen, SunYou, CNE Express, Deppon, YTO, ZTO, 4PX, Cirro, SFC | UPU S10 Mod 11, SF 12-14d, YP suffix, SY format, 3A prefix |
| **2** | **Global & Cross-Border** 🌐 | 595 | Aramex, SpeedPAK, Pitney Bowes, Landmark Global, WANBExpress, Winit, CEVA, DSV, Asendia | EE/ES 16d, UPAAB prefix, WNB prefix, AWB format |
| **3** | **Asia & Oceania (APAC)** 🇯🇵 🇰🇷 🇮🇳 🇦🇺 | 125 | Australia Post, Japan Post, Korea Post, India Post, Singapore Post, NZ Post, Ninja Van, Delhivery, Blue Dart | UPU S10 Mod 11, Ninja Van prefixes, Delhivery barcodes |
| **4** | **Central/Eastern Europe & Balkans** 🇵🇱 🇨🇿 🇷🇴 | 81 | Poczta Polska, InPost Poland, Czech Post, Romania Post, ELTA Greece, Bulgaria Post, Sameday, Fan Courier | UPU S10 Mod 11, InPost 24-digit, Sameday barcodes |
| **5** | **North America** 🇺🇸 🇨🇦 | 42 | USPS, UPS, FedEx, Canada Post, Purolator, Canpar, OnTrac, Intelcom Express, Chit Chats | UPU S10, USPS 20-22d, UPS 1Z Mod 10, Canada 16d, Purolator PIN |
| **6** | **Nordic & Baltic Countries** 🇸🇪 🇳🇴 🇩🇰 🇫🇮 | 23 | PostNord Sweden, Posten Norge, Posti Finland, Omniva Estonia, Latvia Post, Lithuania Post, Iceland Post, Bring, DAO | UPU S10 Mod 11, Posti JJFI, Bring 70-prefix, DAO barcodes |
| **7** | **Spain** 🇪🇸 | 24 | Correos España, Correos Express, CTT Express, SEUR, Paack, MRW, GLS Spain | UPU S10, 16/23d domestic, SEUR 14d, MRW franchise codes |
| **8** | **Global Air Cargo (IATA AWB)** ✈️ | 136 | Emirates SkyCargo, Qatar Airways Cargo, Lufthansa Cargo, Iberia Cargo, Turkish Cargo, British Airways Cargo | IATA AWB `NNN-NNNNNNNN` with Mod 7 checksum & flight legs |
| **9** | **Italy & San Marino** 🇮🇹 🇸🇲 | 19 | Poste Italiane, SDA Express, BRT Bartolini, GLS Italy, Nexive | UPU S10 Mod 11, SDA domestic, BRT 12-digit barcodes |
| **10** | **France & Monaco** 🇫🇷 🇲🇨 | 22 | Colissimo, La Poste, Chronopost France, Colis Privé, Relais Colis, Geodis, Mondial Relay | UPU S10, 6A/6M domestic, Chronopost barcodes, Relais Colis |
| **11** | **United Kingdom & Ireland** 🇬🇧 🇮🇪 | 38 | Royal Mail, Parcelforce, Evri (Hermes UK), Yodel, DPD UK, An Post Ireland, Whistl | UPU S10 Mod 11, Evri 16-digit, Yodel JD prefix, DPD 1550-prefix |
| **12** | **Middle East & Africa (MENA)** 🇦🇪 🇸🇦 🇮🇱 | 57 | Emirates Post, Saudi Post (SPL), Turkey Post (PTT), Israel Post, South Africa Post, Egypt Post, Morocco Post, Naqel | UPU S10 Mod 11, Naqel NQ-prefix, PTT domestic |
| **13** | **Germany & DACH Region** 🇩🇪 🇦🇹 🇨🇭 | 26 | Deutsche Post, DHL Paket, Swiss Post, Austrian Post, Hermes Germany, Trans-o-Flex, DPD Germany, Dachser | UPU S10, Swiss 18d, Deutsche Post 0034043, Hermes H-prefix |
| **14** | **Latin America & Caribbean (LATAM)** 🇲🇽 🇧🇷 🇦🇷 | 25 | Correios Brazil, Correos de México, Correo Argentino, Serpost Peru, Correo Uruguayo, Estafeta, Chilexpress | UPU S10 Mod 11, Estafeta 22-digit waybills |
| **15** | **Ocean Freight & Shipping Containers** 🚢 | 49 | MSC, Maersk Line, CMA CGM, Evergreen, COSCO Shipping, Hapag-Lloyd, ONE, ZIM, Yang Ming, Wan Hai | ISO 6346 4-letter + 7-digit container standard |
| **16** | **Benelux** 🇳🇱 🇧🇪 🇱🇺 | 18 | PostNL, Bpost Belgium, Luxembourg Post, DHL Parcel Netherlands, Budbee, Dynalogic | UPU S10 Mod 11, PostNL 3S barcodes, Bpost 24d |
| **17** | **Portugal** 🇵🇹 | 12 | CTT Portugal, CTT Expresso, Chronopost Portugal, DPD Portugal, Vasp Expresso | UPU S10 Mod 11, CTT domestic |

---

## 2. Checksum Verification Algorithms

`open-parcels` does not just rely on regex pattern matching; it implements standard mathematical checksum verifications:

1. **UPU S10 Modulo 11**:
   - Standard international postal tracking codes: 2 letters, 8 serial digits, 1 check digit, 2 country letters.
   - Weights: `[8, 6, 4, 2, 3, 5, 9, 7]`.
   - Remainder mod 11 calculates check digit (10 ➔ 0, 11 ➔ 5).

2. **UPS 1Z Modulo 10**:
   - Characters 2..16 are converted from alphanumeric digits/letters with alternating weights (1, 2) and summed mod 10 against digit 17.

3. **IATA Air Cargo Modulo 7**:
   - `NNN-NNNNNNNN`: 3-digit airline prefix + 7-digit serial + 1-digit check.
   - Verified via `serial % 7 == check`.

4. **ISO 6346 Container Codes**:
   - 4 alphabetic prefix characters (owner + category U/J/Z) + 6 serial digits + 1 check digit.
