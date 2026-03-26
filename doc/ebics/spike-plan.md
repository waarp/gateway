# Plan de spikes - `amqp091`, `amqp10`, `ebics`, `updateconf` et `Rule`

## 1. Objet

Ce plan de spikes vise a reduire les principaux risques avant tout
developpement d'implementation durable.

Les spikes doivent produire:

- une preuve de faisabilite technique;
- une preuve d'integration avec le modele Gateway;
- une decision `GO / GO sous reserve / NO-GO`.

## 2. Spike A - protocole `amqp091`

### But

Verifier qu'un protocole Gateway `amqp091` natif est simple a introduire et
exploitablement robuste.

### Hypotheses

- `rabbitmq/amqp091-go` suffit pour le besoin nominal;
- les patterns Gateway `Client` / `RemoteAgent` / `ProtoConfig` sont adaptables
  sans contorsion.

### Preuves attendues

- connexion TLS et authentification nominales;
- publication durable avec confirms;
- consommation avec `prefetch` et ack manuel;
- serialisation propre du `ProtoConfig`;
- instrumentation minimale exploitable.

## 3. Spike B - protocole `amqp10`

### But

Verifier qu'un protocole Gateway `amqp10` natif est faisable sans dependance C
et sans dette d'exploitation excessive.

### Hypotheses

- `Azure/go-amqp` est suffisant comme client nominal;
- le protocole couvre proprement le cas des SI enterprise heterogenes.

### Preuves attendues

- connexion TLS nominale;
- emission/reception robustes;
- comportement clair sur ack/reglement des messages;
- gestion exploitable des erreurs et reconnexions;
- serialisation propre du `ProtoConfig`.

## 4. Spike C - `updateconf` et round-trip de configuration

### But

Verifier que l'introduction de nouveaux protocoles ne casse pas la chaine
`export/import/updateconf`.

### Hypotheses

- le modele de backup existant accepte deja de nouveaux protocoles via
  `Protocol` + `ProtoConfig`;
- l'effort principal porte sur la validation, les exemples et la documentation.

### Preuves attendues

- export JSON/YAML contenant `amqp091`, `amqp10`, `ebics`;
- reimport correct via `waarp-gatewayd import`;
- archive ZIP compatible `updateconf`;
- restitution correcte des `ProtoConfig` et des `Rule` techniques EBICS.

### Signal d'alerte

- si un nouveau protocole impose une adaptation structurelle lourde du format
  de backup, il faut reevaluer le cout de la generalisation.

## 5. Spike D - projection EBICS vers `Transfer` et `Rule`

### But

Verifier que le modele `Transfer`/`Rule` ne force pas une deformation
conceptuelle d'EBICS.

### Hypotheses

- seuls les flux EBICS orientes fichier doivent devenir des `Transfer`;
- les ordres administratifs EBICS doivent rester hors `Transfer`.

### Preuves attendues

- un flux EBICS fichier peut etre projete sur `Transfer` avec `Rule`;
- un ordre `HPD` / `HKD` / `HTD` / `HAA` / `INI` / `HIA` peut etre gere sans
  creer de `Transfer`;
- la supervision distingue clairement transfert fichier et operation
  protocolaire;
- l'administration reste comprehensible pour l'exploitant.

### Signal d'alerte

- si EBICS oblige a creer des `Transfer` artificiels pour tout, il y a un vrai
  risque d'inadequation de Gateway.

## 6. Spike E - integration EBICS sur passe-plat asynchrone

### But

Verifier que les payloads et evenements EBICS peuvent etre remis a l'application
metier sans glissement vers de la logique metier.

### Hypotheses

- l'application metier consomme les payloads et decide;
- Gateway remet fichiers, metadonnees et evenements techniques;
- Gateway peut recevoir des metadonnees comme `declaredAmount`.

### Preuves attendues

- emission d'un evenement technique vers `amqp091` ou `amqp10`;
- remise d'un payload ou d'une reference de payload;
- correlation stable entre ordre EBICS, transaction EBICS et evenement sortant;
- reprise sur incident sans duplication non controlee.

## 7. Ordre recommande

1. Spike C - `updateconf`
2. Spike A - `amqp091`
3. Spike B - `amqp10`
4. Spike D - `Transfer` / `Rule` pour EBICS
5. Spike E - integration EBICS sur messaging

Cet ordre est volontaire:

- il ferme d'abord la question du produit complet et exploitable;
- puis il valide le socle messaging;
- ensuite seulement il verifie le couplage fin EBICS/Gateway.

## 8. Criteres de sortie

Le cadrage est valide si:

- `amqp091` et `amqp10` rentrent dans le modele protocolaire Gateway sans
  exception architecturale majeure;
- `updateconf` transporte correctement ces nouveaux protocoles;
- EBICS peut utiliser `Rule` uniquement quand il y a reellement transfert de
  fichier;
- aucun spike ne force Gateway a interpreter les decisions metier.

## 9. Livrables attendus

Pour chaque spike:

- note de resultat courte;
- liste des irritants;
- decision `GO / GO sous reserve / NO-GO`;
- impact sur backlog et architecture.
