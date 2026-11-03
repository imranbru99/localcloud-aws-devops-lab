# AWS & DevOps learning guide

Start the gateway with `./cli/localcloud start` and visit `/console/academy`.
The academy works without third-party fonts, scripts or a frontend build step.
AWS CLI and external practice tools must be installed separately.

## Recommended sequence

1. Foundations: cloud concepts and account safety; Linux/networking/scripting; Git.
2. AWS core: identity/governance; VPC; compute/scaling; storage; databases;
   messaging; serverless; DNS/content delivery.
3. Delivery: containers/ECS; Kubernetes/EKS/GitOps; infrastructure as code; CI/CD.
4. Operations: observability/SRE; DevSecOps; incident response/recovery; FinOps.
5. Specializations: data/analytics; migration, hybrid, AI, IoT and batch orientation.
6. Portfolio: complete the static website, order processing and delivery platform capstones.

Allow roughly four study hours per module, plus tools setup and project work.
Use the in-app planner to estimate the number of weeks at your preferred pace.
Read, explain, practice, troubleshoot, record evidence, then answer the quiz.
Revisit material after a day, a week and a month. Explore each module's listed
subtopics in the official documentation; orientation modules are not deep courses
on every named service.

## Evidence and assessment

A module is complete only when its quiz is passed and the learner explicitly
checks the practice-evidence box. This is self-assessment, not remote verification.
Retaking a quiz incorrectly clears the quiz pass until answered correctly again.
Capstone checklists are also self-assessed. Keep a repository of actual artifacts,
commands/output, diagrams, failure experiments, rollback and restore evidence.
The original lab checker no longer awards success for unchecked resources.
WordPress checks only bucket/database presence, and requires manual verification
of the running application even when those resource checks score 100.

Notes and completion use the browser's `localcloud-academy-v1` localStorage key.
They are not synced, authenticated or stored by the server. Clearing browser data
removes them. Export Markdown from My notebook for a portable record; it is not
an importable progress backup. Never enter secrets in notes.

## Execution boundaries

- Local API: selected operations provided by existing service adapters.
- External tools: Git, Docker, Kubernetes, Terraform/OpenTofu and other tools run
  separately. A local API resource record does not prove an application is running.
- Design / AWS: architectures involving managed services must be implemented and
  verified separately. Use a sandbox account, least privilege, current cost estimates,
  and teardown/retention checks. Budget notifications are not spending caps.

Use `docs/COVERAGE.md` together with adapter source to inspect local support.
Historical coverage descriptions and example billing values are not guarantees
of AWS parity or current pricing. Regional availability, limits and prices change.

## Maintaining the academy

`console/public/curriculum.json` is the versioned content source; it is embedded
and served from `/api/curriculum`. `console/public/academy.html` is a self-contained,
responsive UI embedded at `/console/academy`. Preserve stable module IDs so saved
progress remains associated with the correct lesson. New schema versions require
an explicit storage migration strategy. No learner notes are accepted by the API.

Run `go test ./...` to verify routing, curriculum prerequisite/reference integrity
and that unsupported checks cannot return false passes. Check the academy in a
browser after changes, including search, quizzes, notes, refresh and export.
Rebuild the gateway after changing embedded files.

## Official reading

- [AWS service documentation](https://docs.aws.amazon.com/)
- [Well-Architected Framework](https://docs.aws.amazon.com/wellarchitected/latest/framework/welcome.html)
- [DevOps Guidance](https://docs.aws.amazon.com/wellarchitected/latest/devops-guidance/)
- [Pricing Calculator](https://calculator.aws/)

Developed by [Imran Ahmed](https://imrandev.bd).
