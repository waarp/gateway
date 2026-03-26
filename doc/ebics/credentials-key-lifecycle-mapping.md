# Mapping `Credential` <-> `EbicsKeyLifecycle`

## 1. Objet

Ce document fixe le mapping exact entre:

- le stockage existant de materiel cryptographique dans Gateway via
  `Credential`;
- le workflow dedie `EbicsKeyLifecycle`.

L'objectif est de reutiliser correctement l'existant sans faire porter a
`Credential` une semantique de workflow qu'il ne possede pas.

## 2. Principes

- `Credential` porte le materiel cryptographique;
- `EbicsKeyLifecycle` porte le cycle de vie EBICS de ce materiel;
- un changement de cle EBICS n'est jamais une simple mise a jour destructive
  d'un credential existant;
- l'activation logique d'une cle est portee par le workflow, pas par une seule
  modification en place.

## 3. Repartition des responsabilites

### 3.1 `Credential`

Doit porter:

- le certificat public;
- la cle privee associee quand elle existe;
- le rattachement a l'objet Gateway proprietaire;
- le nom d'administration;
- le type de credential;
- la serialisation et la protection en base.

Ne doit pas porter seul:

- la notion de cle `future`;
- la notion de cle `retired`;
- la date d'activation EBICS;
- la reference d'ordre EBICS de rotation;
- l'evidence operateur.

### 3.2 `EbicsKeyLifecycle`

Doit porter:

- le type de rotation;
- la cle courante;
- la cle future;
- l'etat de workflow;
- l'ordre declencheur;
- les dates de passage d'etat;
- les evidences et confirmations.

## 4. Mapping cible

### 4.1 Cote SQL

`EbicsKeyLifecycle` doit referencer:

- `current_credential_id`
- `next_credential_id`

Recommandation:

- les deux references doivent pointer vers `credentials.id`;
- `next_credential_id` peut etre renseigne tres tot, meme avant emission des
  ordres EBICS;
- le workflow porte la verite de transition entre ces deux credentials.

### 4.2 Cote logique

Cas nominal:

1. un credential courant existe;
2. un nouveau credential est cree comme materiel futur;
3. `EbicsKeyLifecycle` relie les deux;
4. les ordres EBICS sont emis;
5. apres confirmation/activation, le workflow marque le `next` comme actif
   logique;
6. l'ancien credential est conserve pour audit, puis eventuellement retire de
   l'usage.

## 5. Nommage recommande

Pour eviter les ambiguities d'administration:

- les credentials EBICS doivent avoir un nom technique stable;
- le workflow doit porter en plus un alias lisible.

Exemple:

- `ebics-PARTNER01-USER01-auth-current`
- `ebics-PARTNER01-USER01-auth-next`

Le nom de credential ne suffit pas a lui seul a exprimer l'etat reel du cycle
de vie.

## 6. Etats recommandes du workflow

Etats minimaux:

- `DRAFT`
- `MATERIAL_PREPARED`
- `ORDER_PLANNED`
- `ORDER_SENT`
- `WAITING_BANK_CONFIRMATION`
- `ACTIVATED`
- `RETIRED`
- `CANCELLED`
- `REJECTED`

Interpretation:

- `MATERIAL_PREPARED`:
  le `next_credential_id` existe;
- `ACTIVATED`:
  le materiel futur devient la reference active;
- `RETIRED`:
  l'ancien materiel ne doit plus etre utilise.

## 7. Regles de coexistence

Pour un meme usage de cle EBICS:

- un seul lifecycle peut etre actif a la fois;
- un seul credential est `current`;
- un seul credential peut etre `next` en attente;
- plusieurs credentials historiques peuvent subsister pour audit.

## 8. Regles de suppression et retrait

Interdictions:

- ne pas supprimer un credential reference par un lifecycle actif;
- ne pas ecraser `current_credential_id` par update destructive.

Recommandation:

- utiliser un retrait logique avant suppression physique;
- conserver les references historiques pour audit.

## 9. Restitution REST/CLI/UI

Le detail d'un lifecycle doit exposer:

- `currentCredential`
- `nextCredential`
- `keyUsage`
- `rotationType`
- `status`
- `triggerOperation`
- `requestedAt`
- `sentAt`
- `activatedAt`
- `retiredAt`
- `operator`
- `evidence`

Le detail d'un credential utilise en EBICS doit pouvoir afficher:

- `ebicsUsage`
- `lifecycleId`
- `lifecycleRole`:
  - `current`
  - `next`
  - `historical`

## 10. Decision recommandee

Le couple cible est:

- `Credential` comme conteneur de materiel;
- `EbicsKeyLifecycle` comme gouvernance de transition.

C'est la bonne maniere de reutiliser l'existant Gateway sans lui faire porter
une semantique de workflow implicite.
