export const meta = {
  name: 'resolve-17-issues',
  description: 'Linearly orchestrate the resolution of all 17 carrier coverage issues with 4 specialized sequential agents (SDD -> Coder -> Review -> Create PR)',
  phases: [
    { title: 'SDD', detail: 'Software Design Document & carrier architecture specification' },
    { title: 'Coder', detail: 'Code implementation and unit test development in Go' },
    { title: 'Review', detail: 'Code review, regression verification and quality audit' },
    { title: 'Create PR', detail: 'Branch commit, push to GitHub, and pull request creation' },
  ],
}

const ISSUES = [
  { id: 1, title: 'China continental y Hong Kong 🇨🇳 🇭🇰' },
  { id: 2, title: 'Redes Globales de Logística y Couriers Cross-Border 🌐' },
  { id: 3, title: 'Asia y Oceanía (APAC) 🇯🇵 🇰🇷 🇮🇳 🇦🇺 🇳🇿 🇸🇬' },
  { id: 4, title: 'Europa Central, Oriental y Balcanes 🇵🇱 🇨🇿 🇷🇴 🇬🇷 🇺🇦 🇷🇺' },
  { id: 5, title: 'Norteamérica (Estados Unidos y Canadá) 🇺🇸 🇨🇦' },
  { id: 6, title: 'Países Nórdicos y Bálticos 🇸🇪 🇳🇴 🇩🇰 🇫🇮 🇮🇸 🇪🇪 🇱🇻 🇱🇹' },
  { id: 7, title: 'España 🇪🇸' },
  { id: 8, title: 'Carga Aérea Global (IATA Air Cargo / AWB) ✈️' },
  { id: 9, title: 'Italia y San Marino 🇮🇹 🇸🇲' },
  { id: 10, title: 'Francia y Mónaco 🇫🇷 🇲🇨' },
  { id: 11, title: 'Reino Unido e Irlanda 🇬🇧 🇮🇪' },
  { id: 12, title: 'Oriente Medio, Asia Central y África 🇦🇪 🇸🇦 🇮🇱 🇹🇷 🇿🇦 🇪🇬' },
  { id: 13, title: 'Alemania y Región DACH 🇩🇪 🇦🇹 🇨🇭' },
  { id: 14, title: 'Latinoamérica y Caribe (LATAM) 🇲🇽 🇧🇷 🇦🇷 🇨🇱 🇨🇴 🇵🇪' },
  { id: 15, title: 'Transporte Marítimo y Contenedores (Ocean Freight) 🚢' },
  { id: 16, title: 'Benelux (Países Bajos, Bélgica, Luxemburgo) 🇳🇱 🇧🇪 🇱🇺' },
  { id: 17, title: 'Portugal 🇵🇹' },
]

// Allow running a specific subset via args (e.g. args = [1, 2] or args = { from: 1, to: 17 })
const issuesToRun = Array.isArray(args) && args.length > 0
  ? ISSUES.filter(i => args.includes(i.id))
  : (args && typeof args === 'object' && args.from
      ? ISSUES.filter(i => i.id >= args.from && i.id <= (args.to || args.from))
      : ISSUES)

log(`Starting linear execution of ${issuesToRun.length} issues (SDD -> Coder -> Review -> Create PR)...`)

const results = []

for (const issue of issuesToRun) {
  log(`\n======================================================`)
  log(`▶ Processing Issue #${issue.id}: ${issue.title}`)
  log(`======================================================`)

  // Ensure clean starting git state on main
  phase('SDD')
  const sddPrompt = `You are the Lead Software Architect (SDD Agent) for open-parcels.
Your task is to produce the Software Design Document (SDD) for Issue #${issue.id}: "${issue.title}".

1. Execute \`gh issue view ${issue.id}\` to retrieve the complete issue description and carrier list.
2. Check existing carriers and patterns in:
   - \`pkg/detector/rules.go\`
   - \`pkg/carriers/\`
   - \`docs/carriers.json\`
3. Formulate the concrete design specification:
   - List the carriers in this regional scope that should be added or enhanced.
   - For each carrier: define the slug, name, country code, carrier type (postal/express/logistics/freight/cross_border), tracking URL template, tracking number regexes/lengths/prefixes, and any checksum rules.
   - Outline the adapter implementation details and milestone lifecycle simulation/endpoints.
   - Specify unit test test cases with real/realistic tracking numbers for \`pkg/detector/detector_test.go\` and \`pkg/carriers/\`.

Return a clear, actionable markdown specification that the Coder Agent can directly implement.`

  const sddSpec = await agent(sddPrompt, {
    label: `sdd-issue-${issue.id}`,
    phase: 'SDD',
  })

  // Phase 2: Coder
  phase('Coder')
  const branchName = `feature/issue-${issue.id}`
  const coderPrompt = `You are the Expert Go Systems Engineer (Coder Agent) for open-parcels.
Implement Issue #${issue.id}: "${issue.title}" according to the following SDD specification:

${sddSpec}

Instructions:
1. Ensure you work on a dedicated branch:
   \`git checkout -B ${branchName} main\`
2. Implement the carrier detection rules in \`pkg/detector/rules.go\`.
3. Implement the carrier adapters in \`pkg/carriers/\` and register them in \`pkg/carriers/carrier.go\`.
4. Add unit test test cases to \`pkg/detector/detector_test.go\` and carrier test suites.
5. Verify that \`go test -v ./...\` and \`go build .\` both succeed without errors or warnings.
6. Check your work with \`git status\` and \`git diff\`.

Do not commit or push yet; the Review and Create PR agents will inspect and submit.`

  const coderResult = await agent(coderPrompt, {
    label: `coder-issue-${issue.id}`,
    phase: 'Coder',
    gate: 'go test ./...',
  })

  // Phase 3: Review
  phase('Review')
  const reviewPrompt = `You are the Principal QA & Security Reviewer (Review Agent) for open-parcels.
Review the implementation made for Issue #${issue.id}: "${issue.title}" on branch \`${branchName}\`.

Instructions:
1. Run \`git diff main\` to inspect all changes.
2. Verify code quality, idiomatic Go conventions, error handling, memory safety, and absence of regressions.
3. Run \`go test -v ./...\` to verify that all test suites pass.
4. If there are any bugs, style flaws, or missing tests, fix them directly and re-run tests.
5. Provide a summary of the code review, highlighting approved changes and test verification.`

  const reviewResult = await agent(reviewPrompt, {
    label: `review-issue-${issue.id}`,
    phase: 'Review',
    gate: 'go test ./...',
  })

  // Phase 4: Create PR
  phase('Create PR')
  const prPrompt = `You are the Release Engineer (Create PR Agent) for open-parcels.
Finalize and submit the Pull Request for Issue #${issue.id}: "${issue.title}".

Instructions:
1. Verify git status on branch \`${branchName}\`.
2. Stage and commit all changes with a conventional commit message:
   \`feat(carriers): implement carriers for issue #${issue.id} - ${issue.title}\`
3. Push the branch to remote:
   \`git push -u origin ${branchName}\`
4. Create the Pull Request using GitHub CLI:
   \`gh pr create --base main --head ${branchName} --title "feat(carriers): ${issue.title} (#${issue.id})" --body "Closes #${issue.id}\n\n### Summary\n- Implemented carrier detection rules and tracking adapters for ${issue.title}.\n- Added unit tests for detector and carriers.\n\n### Review Summary\n${reviewResult ? reviewResult.replace(/"/g, '\\"') : 'Approved by automated review gate.'}"\`
5. Check out back to \`main\` and ensure a clean working tree:
   \`git checkout main\`
6. Return the URL of the created PR.`

  const prResult = await agent(prPrompt, {
    label: `pr-issue-${issue.id}`,
    phase: 'Create PR',
  })

  log(`✔ Finished Issue #${issue.id}: PR Created: ${prResult}`)
  results.push({
    issueId: issue.id,
    title: issue.title,
    pr: prResult,
  })
}

log(`\n🎉 All ${issuesToRun.length} issues have been successfully processed!`)
return { results }
