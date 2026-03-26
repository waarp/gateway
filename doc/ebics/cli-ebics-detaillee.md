# Commandes CLI detaillees pour EBICS

## 1. Objet

Ce document derive les routes REST EBICS en commandes CLI detaillees, dans le
style nominal de Gateway.

Il fixe:

- la grammaire de commande;
- les options cibles;
- les regles d'affichage;
- les garde-fous d'exploitation lies aux return codes EBICS.

## 2. Principes de conception

- rester proche des patterns existants de `transfer`;
- distinguer clairement `transfer` et `ebics operation`;
- ne pas proposer en CLI une action plus permissive que le protocole;
- conserver la separation `technical` / `business` dans les sorties;
- faire apparaitre la decision Gateway derivee sans masquer les codes source.

## 3. Racine de commande

Famille cible:

```text
waarp-gateway ebics ...
```

Sous-familles de phase 1:

- `payload`
- `operation`
- `contract-view`
- `transaction`
- `rtn event`
- `rtn provider`
- `initialization`
- `key-lifecycle`

## 4. Famille `ebics payload`

Cette famille couvre les ordres EBICS qui manipulent un payload reel:

- `BTU`
- `BTD`
- `FUL`
- `FDL`

Principe:

- la commande soumet une intention protocolaire EBICS;
- Gateway cree une `EbicsOperation`;
- si un fichier est reellement remis ou collecte, Gateway cree ou projette
  aussi un `Transfer`;
- la CLI doit rendre visibles les deux identifiants si les deux objets
  existent.

### 4.1 Soumission d'un upload

Commandes cibles:

```text
waarp-gateway ebics payload upload btu
waarp-gateway ebics payload upload ful
```

Options minimales:

- `--profile`
- `--rule`
- `--resolution-mode`
- `--partner-id`
- `--user-id`
- `--host-id`
- `--file`
- `--output-name`
- `--service-name`
- `--service-option`
- `--scope`
- `--msg-name`
- `--container`
- `--correlation-id`
- `--declared-amount`
- `--requested-date`
- `--output`

Regles:

- `BTU` est privilegie pour les flux BTF cibles;
- `FUL` reste disponible pour les cas historiques ou non-BTF;
- `--profile` doit permettre de charger une configuration EBICS predefinie;
- `--profile` constitue le mode nominal d'exploitation;
- `--rule` reste disponible pour la politique technique Gateway de projection
  fichier;
- les valeurs explicites doivent pouvoir surcharger le profil;
- le mode entierement explicite doit rester reserve aux usages d'administration,
  de diagnostic ou de bootstrap;
- la commande doit afficher au minimum `operationId` et, si cree
  immediatement, `transferId`.

Exemple:

```text
waarp-gateway ebics payload upload btu \
  --partner-id PARTNER01 \
  --user-id USER01 \
  --host-id BANKHOST01 \
  --file C:\payloads\pain001.xml \
  --service-name SCT \
  --service-option COR \
  --scope GLB \
  --msg-name pain.001 \
  --declared-amount 1520.45
```

### 4.2 Soumission d'un download

Commandes cibles:

```text
waarp-gateway ebics payload download btd
waarp-gateway ebics payload download fdl
waarp-gateway ebics payload profile add
waarp-gateway ebics payload profile list
waarp-gateway ebics payload profile get <name>
waarp-gateway ebics payload profile update <name>
```

Options minimales:

- `--profile`
- `--rule`
- `--resolution-mode`
- `--partner-id`
- `--user-id`
- `--host-id`
- `--target-dir`
- `--service-name`
- `--service-option`
- `--scope`
- `--msg-name`
- `--container`
- `--correlation-id`
- `--from-date`
- `--to-date`
- `--output`

Regles:

- `BTD` est privilegie pour les collectes BTF cibles;
- `FDL` reste disponible pour les telechargements de fichiers non BTF;
- `--profile` doit pouvoir eviter la repetition de `serviceName`,
  `serviceOption`, `scope`, `msgName`;
- `--profile` constitue le mode nominal d'exploitation;
- les parametres explicites gardent priorite sur le profil;
- le mode explicite doit rester sous controle operateur et ne pas devenir la
  pratique courante;
- la commande doit afficher `operationId` et `transferId` quand le pipeline
  fichier est projete.

Exemple:

```text
waarp-gateway ebics payload download btd \
  --partner-id PARTNER01 \
  --user-id USER01 \
  --host-id BANKHOST01 \
  --target-dir C:\collecte \
  --service-name CAMT \
  --scope BIL
```

### 4.3 Consultation et suivi

Commandes cibles:

```text
waarp-gateway ebics payload get <operation-id>
waarp-gateway ebics payload list
```

Positionnement:

- ce sont des vues orientees payload;
- elles peuvent etre des alias ergonomiques de `ebics operation get/list`
  filtres sur les ordres `BTU/BTD/FUL/FDL`;
- elles doivent afficher la correlation vers `Transfer`.

### 4.4 Retry et recovery

Commandes cibles:

```text
waarp-gateway ebics payload retry <operation-id>
waarp-gateway ebics payload recover <operation-id>
```

Regles:

- `retry` ne doit etre propose que si la politique de l'ordre l'autorise;
- `recover` doit etre reserve aux cas de transaction/segmentation EBICS;
- la CLI doit expliciter s'il s'agit d'un retry Gateway, d'un retry EBICS ou
  d'une recovery EBICS.

Exemple de message attendu:

```text
Operation 84 requires protocol recovery, not retry: retryDecision=RECOVERY_REQUIRED
```

## 5. Famille `ebics operation`

### 4.1 Liste

Commande cible:

```text
waarp-gateway ebics operation list
```

Options minimales:

- `--limit`
- `--offset`
- `--sort`
- `--status`
- `--operation-type`
- `--order-type`
- `--direction`
- `--partner-id`
- `--user-id`
- `--transaction-id`
- `--request-id`
- `--correlation-id`
- `--transfer-id`
- `--start`
- `--stop`
- `--output`

Valeurs de tri cibles:

- `start+`
- `start-`
- `id+`
- `id-`
- `status+`
- `status-`
- `orderType+`
- `orderType-`

Exemples:

```text
waarp-gateway ebics operation list --status FAILED --order-type HPD
waarp-gateway ebics operation list --partner-id PARTNER01 --start 2026-03-25T00:00:00Z
```

### 4.2 Consultation

Commande cible:

```text
waarp-gateway ebics operation get <id>
```

Sortie humaine attendue:

- identite de l'operation;
- type et ordre EBICS;
- statut;
- identites `host/partner/user`;
- `transactionId`, `requestId`, `correlationId`;
- scope `technical`;
- scope `business`;
- `gatewayOutcome`;
- `retryDecision`;
- `manualActionRequired`;
- lien vers `Transfer` si present.

Exemple d'affichage cible:

```text
=== EBICS Operation ===
-Operation 42 (HPD / CONTRACT_REFRESH) [COMPLETED]
  -Host ID: BANKHOST01
  -Partner ID: PARTNER01
  -User ID: USER01
  -Transaction ID: N/A
  -Request ID: req-123
  -Correlation ID: corr-123
  -Technical return code: 000000
  -Technical message: [EBICS_OK] OK
  -Business return code: 000000
  -Business message: [EBICS_OK] OK
  -Gateway outcome: SUCCESS
  -Retry decision: NO_RETRY
  -Manual action required: no
  -Transfer: N/A
```

### 4.3 Retry

Commande cible:

```text
waarp-gateway ebics operation retry <id>
```

Options minimales:

- `--reason`
- `--meta key:value`

Regles:

- la commande doit echouer si `retryDecision` n'autorise pas l'action;
- la commande doit expliciter s'il s'agit d'un `retry`, d'un `replay` manuel
  ou d'une `recovery` requise;
- la CLI ne doit jamais masquer un refus du serveur lie au scope de return
  codes.

Exemple:

```text
waarp-gateway ebics operation retry 42 --reason "temporary bank outage"
```

### 4.4 Cancel

Commande cible:

```text
waarp-gateway ebics operation cancel <id>
```

Regles:

- reserve aux statuts annulables;
- l'action doit etre presentee comme technique, jamais comme annulation
  metier.

### 4.5 Confirm

Commande cible:

```text
waarp-gateway ebics operation confirm <id>
```

Options minimales:

- `--reason`
- `--meta key:value`

Usage cible:

- confirmation d'une activation externe;
- validation technique de rupture d'automatisme;
- jamais approbation metier riche.

## 6. Famille `ebics contract-view`

### 5.1 Consultation

Commandes cibles:

```text
waarp-gateway ebics contract-view get --partner-id <partner>
waarp-gateway ebics contract-view capabilities --partner-id <partner>
waarp-gateway ebics contract-view permissions --partner-id <partner>
```

### 5.2 Rafraichissement

Commande cible:

```text
waarp-gateway ebics contract-view refresh --partner-id <partner>
```

Resultat attendu:

- creation ou planification d'une `EbicsOperation`;
- affichage de `operationId`.

## 7. Famille `ebics transaction`

Commandes cibles:

```text
waarp-gateway ebics transaction list
waarp-gateway ebics transaction get <transaction-id>
waarp-gateway ebics transaction segments <transaction-id>
waarp-gateway ebics transaction segment get <transaction-id> <segment-number>
```

Positionnement:

- famille orientee diagnostic;
- ne pas la melanger avec les commandes de `transfer`.

## 8. Famille `ebics rtn event`

Commandes cibles:

```text
waarp-gateway ebics rtn event list
waarp-gateway ebics rtn event get <id>
waarp-gateway ebics rtn event retry <id>
waarp-gateway ebics rtn event quarantine <id>
```

Options de liste minimales:

- `--status`
- `--provider`
- `--partner-id`
- `--user-id`
- `--correlation-id`
- `--start`
- `--stop`

## 9. Famille `ebics rtn provider`

Commandes cibles:

```text
waarp-gateway ebics rtn provider list
waarp-gateway ebics rtn provider get <name>
waarp-gateway ebics rtn provider add
waarp-gateway ebics rtn provider update <name>
```

Regle:

- en phase 1, le provider principal vise `WebSocket/WSS`;
- la CLI ne doit pas enfermer le design dans un unique transport.

## 10. Famille `ebics initialization`

Commandes cibles:

```text
waarp-gateway ebics initialization list
waarp-gateway ebics initialization get <id>
waarp-gateway ebics initialization add
waarp-gateway ebics initialization confirm <id>
waarp-gateway ebics initialization cancel <id>
```

Sorties attendues:

- etat du workflow;
- presence de la lettre EBICS;
- etat d'activation banque;
- operation source associee.

## 11. Famille `ebics key-lifecycle`

Commandes cibles:

```text
waarp-gateway ebics key-lifecycle list
waarp-gateway ebics key-lifecycle get <id>
waarp-gateway ebics key-lifecycle add
waarp-gateway ebics key-lifecycle confirm <id>
waarp-gateway ebics key-lifecycle cancel <id>
```

## 12. Regles d'affichage des return codes

La CLI doit toujours afficher separement:

- `Technical return code`
- `Technical message`
- `Business return code`
- `Business message`
- `Gateway outcome`
- `Retry decision`

La CLI ne doit pas:

- afficher un unique `Return code`;
- laisser croire qu'un business reject est un incident reseau;
- proposer `retry` sans indiquer si l'action est autorisee.

## 13. Regles d'ergonomie

Les commandes doivent rester memorisables.

Recommandations:

- garder des verbes simples: `list`, `get`, `retry`, `cancel`, `confirm`,
  `refresh`, `quarantine`;
- conserver les options de date en RFC3339;
- proposer `--output json|yaml|text` comme sur les autres familles;
- preferer `--partner-id` et `--user-id` a des abreviations EBICS trop
  cryptiques.

## 14. Garde-fous d'exploitation

La CLI doit prevenir explicitement l'operateur quand:

- `retryDecision=NO_RETRY`;
- `retryDecision=MANUAL_CONFIRMATION_ONLY`;
- `gatewayOutcome=BUSINESS_REJECTED`;
- un code `technical` de type anti-rejeu ou recovery impose une autre action.
- la commande payload est lancee sans `--profile` alors qu'un profil valide est
  disponible;
- les champs explicites fournis ne correspondent a aucune ligne du contrat
  connu.

Exemples de messages cibles:

- `Operation 42 cannot be retried automatically: retryDecision=NO_RETRY`
- `Operation 42 requires protocol recovery, not retry: retryDecision=RECOVERY_REQUIRED`
- `Operation 42 was rejected by bank business controls: businessReturnCode=091303`

## 15. Decision recommandee

La CLI EBICS doit etre introduite comme une famille parallele aux commandes
`transfer`, sans chercher a les fusionner.

La famille prioritaire de phase 1 est:

- `ebics payload`
- `ebics operation`

car ce sont elles qui portent la plus forte valeur d'exploitation et la plus
forte sensibilite au traitement correct des return codes.
