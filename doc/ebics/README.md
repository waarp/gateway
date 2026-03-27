# Dossier d'integration EBICS dans Waarp Gateway

Ce dossier contient une premiere base de travail pour l'integration de la
librairie WAARP EBICS dans Waarp Gateway.

Documents disponibles:

- `specifications-fonctionnelles.md`
- `specifications-techniques.md`
- `sql-modele-detaille-ebics.md`
- `architecture-logicielle.md`
- `adr-amqp-et-ebics.md`
- `amqp-protocols-architecture.md`
- `amqp-protocols-backlog.md`
- `aptitude-gateway-rotation-cles.md`
- `aptitude-gateway-signatures.md`
- `backend-consolidation-plan.md`
- `backlog-implementation.md`
- `backlog-implementation-fichier-par-fichier.md`
- `catalogue-btf-standard.md`
- `catalogue-btf-standard-design.md`
- `cli-ui-impact-ebics.md`
- `cli-ebics-detaillee.md`
- `decision-gateway-vs-scratch.md`
- `ebics-operations-design.md`
- `ebics-operations-admin-design.md`
- `evaluation-candidate-gateway.md`
- `ergonomie-profils-payload-ebics.md`
- `frontiere-protocole-metier.md`
- `lots-1-2-detail.md`
- `matrice-ordres-ebics.md`
- `mapping-sql-model-rest-ebics.md`
- `routes-rest-ebics-detaillees.md`
- `note-maxamount-et-prevalidation.md`
- `payload-profiles-design.md`
- `payload-submission-contracts.md`
- `phase-a-cadrage-concret.md`
- `phase-a-structs-et-signatures.md`
- `phase-b-cadrage-concret.md`
- `phase-b-structs-et-signatures.md`
- `phase-c-cadrage-concret.md`
- `phase-c-structs-et-signatures.md`
- `phase-d-cadrage-concret.md`
- `phase-d-structs-et-signatures.md`
- `phase-e-cadrage-concret.md`
- `phase-e-structs-et-signatures.md`
- `note-filewatcher-client-lourd.md`
- `points-attention-gateway-ebics.md`
- `proto-config-targets.md`
- `spike-c-updateconf.md`
- `spike-d-transfer-rule-ebics.md`
- `spike-plan.md`
- `suivi-implementation-ebics.md`
- `suivi-implementation-phases.md`
- `suivi-backend-consolidation.md`
- `positionnement-cible.md`
- `returncodes-ebics-gateway.md`
- `credentials-key-lifecycle-mapping.md`
- `signature-states-and-rest.md`
- `impact-existant-credential.md`
- `squelette-technique-ebics.md`

Objectif de cette premiere version:

- repartir de l'existant Gateway plutot que d'introduire une stack parallele,
- identifier ce qui peut etre mappe directement sur le modele courant,
- isoler les extensions strictement necessaires au protocole EBICS,
- definir une architecture permettant l'automatisation maximale tout en
  preservant performance, tracabilite et securite.

Principes de cadrage retenus:

- EBICS est integre comme nouveau protocole natif de Gateway;
- la librairie `code.waarp.fr/lib/ebics` reste proprietaire du protocole EBICS;
- Gateway reste proprietaire de l'hebergement, de la persistance durable, de
  l'administration, de l'observabilite, des droits et de l'orchestration
  technique;
- l'application metier reste proprietaire des decisions non protocolaires et
  des workflows bancaires;
- les traitements EBICS doivent s'aligner sur les standards d'exploitation de
  Gateway: services, base de donnees, REST, CLI, UI, historique, supervision.

Points de focus complementaires integres a cette version:

- gestion du cycle de vie et de la rotation des cles EBICS;
- gestion de la rupture d'automatisme pendant l'initialisation EBICS, en
  particulier autour de la lettre EBICS et de l'activation hors bande;
- gestion des notifications temps reel RTN, comme nouvelle capacite transverse
  absente aujourd'hui de Gateway;
- gestion des signatures au sens protocolaire, sans embarquer le workflow
  metier de signature;
- mise en passe-plat vers l'application metier pour les decisions non
  protocolaires.

Fichiers d'exemple utiles:

- `pkg/backup/testdata/ebics-standard-btf-curated.json`
- `pkg/backup/testdata/ebics-standard-btf-curated.yaml`
