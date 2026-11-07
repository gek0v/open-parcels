#!/usr/bin/env python3
import json
import re
import subprocess
import sys

def main():
    with open('/home/hermes/Workspaces/open-parcels/docs/GITHUB_ISSUES_PLAN.md', 'r', encoding='utf-8') as f:
        content = f.read()

    issues_raw = re.split(r'### ISSUE #\d+:\s*', content)[1:]
    print(f"Encontrados {len(issues_raw)} issues para publicar...")

    # Fetch currently open issues to avoid duplicates
    existing_res = subprocess.run(
        ['gh', 'issue', 'list', '--repo', 'gek0v/open-parcels', '--json', 'title,url', '--limit', '100'],
        capture_output=True, text=True
    )
    existing_map = {}
    if existing_res.returncode == 0:
        for item in json.loads(existing_res.stdout):
            existing_map[item['title']] = item['url']

    created_issues = []

    milestone_titles = {
        "1": "Milestone 1: Core Engine & Standards",
        "2": "Milestone 2: Tier 1 Carriers - España e Iberia",
        "3": "Milestone 3: Tier 1 Carriers - Global & Cross-Border",
        "4": "Milestone 4: Heuristic Detector & Rules Optimization",
        "5": "Milestone 5: MCP, REST API & Async Polling",
        "6": "Milestone 6: Governance, Documentation & Transparency",
    }

    for idx, raw in enumerate(issues_raw, 1):
        section = raw.split('\n---\n')[0].strip()
        lines = section.split('\n')
        title = lines[0].strip()

        if title in existing_map:
            url = existing_map[title]
            print(f"Issue {idx} ya existe: {url}")
            created_issues.append({"idx": idx, "title": title, "url": url})
            continue

        milestone_m = re.search(r'\*\*Milestone:\*\*\s*Milestone\s*(\d+)', section)
        ms_num = milestone_m.group(1) if milestone_m else None
        ms_title = milestone_titles.get(ms_num)

        labels_m = re.search(r'\*\*Etiquetas:\*\*\s*(.*)', section)
        labels = []
        if labels_m:
            labels = re.findall(r'`([^`]+)`', labels_m.group(1))

        area_m = re.search(r'\*\*Área:\*\*\s*(.*)', section)
        area = area_m.group(1).strip() if area_m else ''

        prio_m = re.search(r'\*\*Prioridad:\*\*\s*(.*)', section)
        prio = prio_m.group(1).strip() if prio_m else ''

        body_idx = section.find('#### Contexto y Justificación')
        body_content = section[body_idx:] if body_idx != -1 else section

        full_body = f"> **Área:** {area}\n> **Prioridad:** {prio}\n\n{body_content}"

        cmd = [
            'gh', 'issue', 'create',
            '--repo', 'gek0v/open-parcels',
            '--title', title,
            '--body', full_body
        ]

        if ms_title:
            cmd.extend(['--milestone', ms_title])

        for lbl in labels:
            cmd.extend(['--label', lbl])

        print(f"Publicando Issue {idx}: {title}...")
        res = subprocess.run(cmd, capture_output=True, text=True)
        if res.returncode != 0:
            print(f"Aviso al publicar issue {idx}: {res.stderr}. Reintentando sin etiquetas conflictivas...")
            retry_cmd = [
                'gh', 'issue', 'create',
                '--repo', 'gek0v/open-parcels',
                '--title', title,
                '--body', full_body
            ]
            if ms_title:
                retry_cmd.extend(['--milestone', ms_title])
            res2 = subprocess.run(retry_cmd, capture_output=True, text=True)
            if res2.returncode == 0:
                url = res2.stdout.strip()
                print(f"  -> Creado: {url}")
                created_issues.append({"idx": idx, "title": title, "url": url})
            else:
                print(f"Error persistente en issue {idx}: {res2.stderr}", file=sys.stderr)
                sys.exit(1)
        else:
            url = res.stdout.strip()
            print(f"  -> Creado: {url}")
            created_issues.append({"idx": idx, "title": title, "url": url})

    print(f"\n¡Todos los {len(created_issues)} issues procesados correctamente!")
    with open('/home/hermes/Workspaces/open-parcels/docs/published_issues.json', 'w', encoding='utf-8') as f:
        json.dump(created_issues, f, indent=2, ensure_ascii=False)

if __name__ == '__main__':
    main()
