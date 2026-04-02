# Parametrage RTN WSS et auto-pull EBICS

## 1. Objet

Ce document decrit comment Waarp Gateway determine quel ordre EBICS payload
declencher lorsqu'une banque notifie, via `WSS` (`RTN`), la mise a disposition
d'un document ou d'un acquittement.

Il sert de base pour la documentation utilisateur et pour le cadrage du contrat
attendu entre:

- la banque ou son provider `WSS`;
- la configuration `RTN` de Gateway;
- les profils payload et les contrats EBICS actifs.

## 2. Principe general

Gateway ne "devine" pas seule quel ordre ou quel `BTF` utiliser.

La determination se fait en trois etapes:

1. lecture des metadonnees portees par l'evenement `RTN/WSS`;
2. completion par le profil payload Gateway, si necessaire;
3. validation finale contre le contrat EBICS actif et les regles Gateway.

L'auto-pull n'est donc fiable que si la notification `RTN` fournit assez de
contexte pour identifier sans ambiguite:

- l'ordre payload cible;
- le profil payload a utiliser;
- le service/BTF vise.

## 3. Chaine de decision

### 3.1 Evenement RTN recu

Lors de la reception de l'evenement:

- Gateway normalise le message `RTN`;
- extrait les champs techniques utiles;
- calcule une cle d'idempotence;
- persiste l'evenement avant toute tentative d'auto-pull.

Les champs les plus importants pour l'auto-pull sont:

- `orderTypeHint`
- `profileID`
- `serviceName`
- `serviceOption`
- `scope`
- `msgName`
- `containerType`
- `ruleID` ou `ruleName`
- `targetDirectory`
- `requestID`

### 3.2 Construction du plan d'auto-pull

Gateway construit ensuite un plan d'auto-pull:

- si `orderTypeHint` est present, il est utilise;
- sinon, Gateway retombe par defaut sur `BTD`;
- si `profileID` est present, il designe explicitement le profil payload;
- si le provider est configure en `AUTO_FILTERED` et que `profileID` manque,
  l'auto-pull est rejete.

### 3.3 Resolution payload

Gateway reconstruit une demande payload interne a partir:

- des champs de l'evenement `RTN`;
- des valeurs par defaut du profil payload;
- des regles Gateway resolues.

Le profil payload peut completer:

- l'ordre si necessaire;
- le service;
- la regle par defaut;
- la cible locale.

### 3.4 Validation contractuelle

Avant toute programmation du transfert, Gateway valide la resolution contre:

- la vue contractuelle active de la banque;
- puis, si applicable, le catalogue standard.

Si le couple `ordre + service + message + conteneur` n'est pas autorise,
l'auto-pull est rejete.

## 4. Tableau des champs attendus

| Champ RTN/WSS | Usage dans Gateway | Niveau attendu |
|---|---|---|
| `orderTypeHint` | Indique l'ordre EBICS payload a executer. S'il est absent, Gateway retombe sur `BTD`. | Recommande, quasi obligatoire |
| `profileID` | Designe explicitement le profil payload Gateway a utiliser. C'est le moyen le plus propre pour obtenir un comportement deterministe. | Obligatoire en `AUTO_FILTERED`, fortement recommande en `AUTO` |
| `serviceName` | Participe a l'identification du service/BTF vise. | Recommande |
| `serviceOption` | Affine le service EBICS. | Optionnel selon les cas |
| `scope` | Affine la resolution du service/BTF. | Optionnel selon les cas |
| `msgName` | Identifie le type de message metier attendu. | Recommande |
| `containerType` | Affine la resolution si la banque utilise cette dimension. | Optionnel |
| `ruleID` | Force la regle Gateway a utiliser pour le transfert auto-declenche. | Optionnel |
| `ruleName` | Alternative lisible a `ruleID` pour forcer la regle Gateway. | Optionnel |
| `clientID` | Designe explicitement le client Gateway EBICS a utiliser pour executer l'auto-pull. Cette reference est portee par le provider RTN configure, pas par l'evenement recu. | Obligatoire si la politique est `AUTO` ou `AUTO_FILTERED` |
| `targetDirectory` | Definit la cible locale de reception. | Optionnel |
| `requestID` | Sert d'identifiant/correlation pour l'operation EBICS. | Recommande |
| `hostID` / `partnerID` / `userID` | Contexte abonne EBICS. En pratique, Gateway le connait deja via le provider `RTN` configure. | Optionnel si deja porte par la configuration du provider |

## 5. Regles fonctionnelles importantes

- Si `profileID` est present, Gateway cherche d'abord a resoudre l'auto-pull
  via ce profil.
- Si `orderTypeHint` est absent, Gateway part par defaut sur `BTD`.
- Si le provider est en mode `AUTO_FILTERED` et que `profileID` manque,
  l'auto-pull est refuse.
- La presence d'un profil payload ne dispense jamais de la validation
  contractuelle.
- Si la resolution reste ambigue ou non autorisee contractuellement,
  l'auto-pull echoue proprement.

## 6. Recommandation de contrat minimal cote banque / provider WSS

Pour un auto-pull robuste et deterministe, il faut au minimum transmettre:

- `orderTypeHint`
- `profileID`

Et idealement aussi:

- `serviceName`
- `msgName`
- `scope` si pertinent
- `requestID`

## 7. Cas pratiques

### 7.1 Cas robuste

Notification `RTN` contenant:

- `orderTypeHint=BTD`
- `profileID=bank-camt054-download`
- `serviceName=MCT`
- `msgName=camt.054`

Effet:

- Gateway identifie immediatement le profil payload;
- reconstruit le service cible;
- valide le contrat;
- programme un vrai `Transfer` Gateway pour recuperer le document via EBICS.

### 7.2 Cas fragile

Notification `RTN` contenant seulement:

- "un document est disponible"

Sans:

- `profileID`
- `orderTypeHint`
- service detaille

Effet:

- Gateway ne peut pas choisir proprement le bon ordre/BTF;
- soit l'auto-pull est rejete;
- soit il repose sur un defaut (`BTD`) insuffisant pour garantir le bon
  comportement.

## 8. Conclusion

Pour savoir quel ordre/BTF parametrer apres une notification `RTN/WSS`,
Gateway a besoin d'un contrat de notification suffisamment riche.

Le schema cible est:

- le message `RTN` apporte le contexte;
- le profil payload Gateway complete si necessaire;
- le contrat EBICS valide la legitimite de l'echange.

En pratique, la meilleure strategie est donc:

- de faire porter au message `RTN` un `orderTypeHint`;
- d'y joindre un `profileID`;
- et de transmettre les dimensions de service utiles (`serviceName`,
  `msgName`, `scope`, `containerType`) quand elles ont une valeur
  discriminante.

## 9. Messages d'erreur typiques en cas d'information insuffisante

Si l'evenement `RTN` ne fournit pas assez d'information pour resoudre un
auto-pull exploitable, Gateway enregistre un echec explicite sur l'evenement
`RTN`.

Les informations suivantes sont alors mises a jour:

- le statut de l'evenement;
- le message `LastError`;
- eventuellement une date de retry si l'erreur est consideree comme
  retryable.

### 9.1 Exemples de messages d'erreur attendus

| Cas | Message typique |
|---|---|
| `AUTO_FILTERED` sans `profileID` | `the RTN auto-pull policy AUTO_FILTERED requires a profile reference on the event` |
| ordre payload non resolvable | `the EBICS payload order type is missing` |
| profil payload desactive | `the EBICS payload profile "<nom>" is disabled` |
| aucune regle Gateway resolue | `no Gateway rule could be resolved for the RTN auto-pull` |
| plusieurs regles Gateway candidates | `multiple Gateway rules match the RTN auto-pull request` |
| aucun client EBICS configure | `no EBICS client is configured for the RTN auto-pull` |
| `clientID` provider absent | `the RTN provider client ID is missing` |
| client RTN inconnu, desactive ou non EBICS | message de validation du provider RTN |
| resolution contractuelle invalide | message issu de la validation contractuelle active |

### 9.2 Effet fonctionnel

- si l'erreur est une erreur de validation, l'evenement passe en general en
  `FAILED`;
- sinon, il peut passer en `RETRYABLE`;
- dans les deux cas, le detail est conserve dans `LastError`.

### 9.3 Recommandation utilisateur

Si l'auto-pull `RTN` est active, il faut verifier que la notification fournit
au minimum:

- un `orderTypeHint`;
- un `profileID` si le provider est en `AUTO_FILTERED`;
- qu'un `clientID` explicite est configure sur le provider RTN si la politique
  est `AUTO` ou `AUTO_FILTERED`;
- et idealement les dimensions de service utiles (`serviceName`, `msgName`,
  `scope`, `containerType`).

Sans ces informations, l'auto-pull risque d'echouer de maniere deterministe,
avec un message d'erreur persiste sur l'evenement `RTN`.

## 10. Diagnostic operateur

En phase d'exploitation, le diagnostic d'un auto-pull `RTN` en erreur repose
principalement sur l'evenement `RTN` lui-meme.

Les informations utiles a consulter sont:

- le statut de l'evenement;
- `LastError`;
- le nombre de tentatives;
- la prochaine date de retry, si applicable;
- les informations d'auto-pull derivees (`operation`, `transfer`, `status`,
  `outcome`, `retry`).

### 10.1 Cote REST / CLI

Le diagnostic doit se faire via les surfaces `RTN` REST/CLI de Gateway:

- consultation du detail de l'evenement `RTN`;
- lecture du message `LastError`;
- verification des champs `autoPullOperationID`, `autoPullTransferID`,
  `autoPullStatus`, `autoPullOutcome`, `autoPullRetry`.

### 10.2 Cote utilisateur

Pour un utilisateur ou un exploitant, la lecture cible est la suivante:

- si l'evenement est en `FAILED`, il faut corriger le paramettrage ou le
  contenu de la notification avant nouveau traitement;
- si l'evenement est en `RETRYABLE`, il faut verifier si l'erreur est
  transitoire ou si elle masque un probleme de parametrage;
- si un `autoPullOperationID` existe mais pas de resultat final, il faut
  suivre l'operation EBICS et le transfert associe.

### 10.3 Note UI

Cette note couvre pour l'instant le diagnostic via REST/CLI.

La presentation cible cote interface utilisateur devra etre ajoutee
ulterieurement, avec:

- une formulation plus orientee exploitation;
- une mise en avant des erreurs de parametrage les plus probables;
- un guidage clair entre evenement `RTN`, operation EBICS et transfert
  Gateway.
