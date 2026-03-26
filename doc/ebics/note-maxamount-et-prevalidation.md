# Note sur `MaxAmount` et la prevalidation

## 1. Objet

Cette note precise comment traiter les informations de type `MaxAmount`,
`AuthorisationLevel` et `PreValidation` dans l'integration EBICS de Gateway.

L'objectif est:

- de renforcer la securisation des echanges;
- d'eviter d'ouvrir inutilement les payloads metier dans Gateway;
- de conserver une frontiere claire entre protocole et metier.

## 2. Ce que dit la specification

La specification EBICS 3.0.2 montre que:

- `HKD` / `HTD` peuvent publier des permissions comportant
  `AdminOrderType`, `Service`, `AccountID`, `AuthorisationLevel` et
  `MaxAmount`;
- `HPD` publie notamment le support de `PreValidation`;
- la prevalidation est une capacite optionnelle de la banque, executee cote
  banque sur la base d'informations fournies au premier pas de transaction;
- la prevalidation ne remplace pas les verifications finales faites par la
  banque sur les donnees effectivement transmises.

Conclusion:

- `MaxAmount` est d'abord un fait technique publie par la banque;
- la banque reste l'autorite finale de controle;
- Gateway n'a pas a devenir, par defaut, un moteur d'analyse des payloads
  metier.

## 3. Position retenue pour Gateway

### 3.1 Ce que Gateway doit faire

Gateway doit:

- stocker `MaxAmount` dans la vue contractuelle technique;
- exposer cette information aux operateurs et aux applications tierces;
- l'utiliser pour prevenir et documenter;
- l'utiliser dans des controles preventifs quand une information de montant est
  fournie explicitement a Gateway.

### 3.2 Ce que Gateway ne doit pas faire par defaut

Gateway ne doit pas, par defaut:

- ouvrir les payloads metier pour recalculer les montants;
- parser des formats bancaires metier uniquement pour repliquer un controle
  banque;
- faire de la verification de limite le centre de gravite de la couche
  protocolaire.

## 4. Modele recommande

Le modele recommande est le suivant:

- l'application metier construit ou connait deja l'ordre;
- elle transmet a Gateway les metadonnees techniques necessaires a un controle
  preventif;
- parmi ces metadonnees peut figurer un `declaredAmount`;
- Gateway compare ce `declaredAmount` avec la vue contractuelle technique
  connue;
- la banque effectue ensuite le controle autoritatif final.

## 5. Niveaux de controle possibles

### 5.1 Niveau 0 - Aucune verification locale

Gateway:

- stocke `MaxAmount`;
- journalise la permission;
- laisse la banque faire tout le controle.

Usage:

- mode le plus simple;
- securite minimale cote exploitation.

### 5.2 Niveau 1 - Verification preventive par metadonnees

Gateway:

- recoit un montant declare dans la demande;
- compare ce montant avec `MaxAmount` lorsque la permission applicable est
  connue;
- bloque ou alerte si l'ordre est manifestement hors enveloppe.

Usage:

- compromis recommande;
- forte valeur sans ouvrir les payloads.

Exemple de metadonnees techniques:

```json
{
  "hostId": "BANKFRPP",
  "partnerId": "PARTNER01",
  "userId": "USR200",
  "adminOrderType": "BTU",
  "service": {
    "serviceName": "SCT",
    "scope": "DE",
    "msgName": "pain.001"
  },
  "accountId": "accid01",
  "declaredAmount": {
    "value": "5400.00",
    "currency": "EUR"
  }
}
```

### 5.3 Niveau 2 - Verification locale par ouverture du payload

Gateway:

- parse le payload;
- extrait le montant;
- applique le controle localement.

Usage:

- uniquement si un cas client le justifie fortement;
- uniquement pour un sous-ensemble tres borne et stable de formats;
- a traiter comme une extension optionnelle et non comme la baseline.

Risque:

- derive vers la logique metier;
- couplage fort aux formats bancaires;
- maintenance et responsabilite accrues.

## 6. Decision d'architecture

La baseline recommandee pour Gateway est:

- `Niveau 1 - Verification preventive par metadonnees`.

Donc:

- la vue contractuelle technique stocke `MaxAmount`;
- les API d'emission d'ordre peuvent accepter un montant declare optionnel;
- Gateway peut bloquer, avertir ou journaliser avant emission;
- le controle final reste a la banque;
- l'interpretation profonde du payload reste hors Gateway par defaut.

## 7. Consequences sur les API

Les commandes ou API d'emission d'ordre EBICS devraient pouvoir accepter, de
maniere optionnelle:

- `declaredAmount.value`
- `declaredAmount.currency`
- `declaredAccountId`
- `declaredService`

Ces donnees sont:

- des metadonnees de controle preventif;
- non une substitution au controle protocolaire ou bancaire.

## 8. Criteres de bonne utilisation

Cette approche est bonne si:

- l'application metier connait deja les montants;
- la Gateway reste agnostique du contenu fonctionnel profond;
- les rejets evitables sont reduits;
- l'exploitation gagne en securite sans glissement metier.

Il faudra reouvrir la decision si:

- les clients demandent une verification locale forte sur de nombreux formats;
- les metadonnees transmises par le metier ne sont pas fiables ou pas
  disponibles;
- la valeur de la verification locale depasse clairement son cout
  d'integration.
