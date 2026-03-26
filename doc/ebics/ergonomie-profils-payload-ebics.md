# Ergonomie des profils payload EBICS

## 1. Objet

Ce document fixe une ligne d'ergonomie pour la soumission des ordres payload
EBICS (`BTU`, `BTD`, `FUL`, `FDL`).

Le probleme vise est simple:

- certains attributs sont obligatoires pour construire correctement l'enveloppe
  EBICS;
- leur saisie brute a chaque commande est lourde et source d'erreur;
- `Rule` cree de la valeur pour le routage technique Gateway, mais ne doit pas
  devenir la seule abstraction de la semantique BTF/EBICS.

## 2. Constats

Pour certains flux, l'operateur doit connaitre et renseigner:

- `serviceName`
- `serviceOption`
- `scope`
- `msgName`
- `declaredAmount` dans les cas utiles

Ces donnees sont:

- indispensables au protocole EBICS;
- souvent stables pour une famille de flux;
- peu ergonomiques si on les exige a chaque fois en ligne de commande.

## 3. Decision recommandee

Introduire une notion dediee de profil predefini, par exemple:

- `EbicsPayloadProfile`

Ce profil porte la semantique de soumission EBICS reutilisable.

`Rule` conserve son role:

- routage technique Waarp;
- pipeline fichier;
- repertoire source/destination;
- taches techniques.

Le profil EBICS porte:

- le type d'ordre cible (`BTU`, `BTD`, `FUL`, `FDL`);
- les parametres de service BTF/ordre;
- les defaults de construction de requete;
- les validations d'usage.

## 4. Repartition recommandee

### 4.1 Dans `EbicsPayloadProfile`

- `profileName`
- `orderType`
- `serviceName`
- `serviceOption`
- `scope`
- `msgName`
- `container`
- `defaultDeclaredCurrency` si besoin
- `requiresDeclaredAmount`
- `downloadTargetKind`
- `allowedDirections`
- `contractViewConstraint`
- `description`

### 4.2 Dans `Rule`

- chemin/routage de fichiers;
- pre/post tasks;
- politique de remise vers le metier;
- options de stockage local;
- conventions de nommage locales.

## 5. Relation entre profil et `Rule`

Le bon compromis est:

- un profil payload EBICS peut etre reference seul;
- une `Rule` peut etre referencee seule si tous les parametres EBICS sont
  fournis explicitement;
- une commande peut referencer les deux.

Usage recommande:

- `profile` pour la semantique EBICS;
- `rule` pour la technique Gateway.

Autrement dit:

- le profil ne remplace pas `Rule`;
- `Rule` ne remplace pas le profil.

## 6. Resolution des parametres

Ordre de priorite recommande:

1. valeurs explicites de la commande ou de l'appel API;
2. valeurs du `EbicsPayloadProfile`;
3. defaults techniques derives du contexte Gateway.

Regle importante:

- une valeur explicite doit toujours pouvoir surcharger un profil.

Ainsi:

- l'ergonomie est amelioree;
- l'exploitant garde la main;
- le diagnostic reste possible.

## 7. Ergonomie CLI

Au lieu d'ecrire a chaque fois:

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

on doit pouvoir ecrire:

```text
waarp-gateway ebics payload upload \
  --profile sct-corp-credit-transfer \
  --file C:\payloads\pain001.xml \
  --declared-amount 1520.45
```

ou:

```text
waarp-gateway ebics payload upload \
  --profile sct-corp-credit-transfer \
  --rule ebics-send-default \
  --file C:\payloads\pain001.xml
```

et conserver la possibilite de surcharge:

```text
waarp-gateway ebics payload upload \
  --profile sct-corp-credit-transfer \
  --service-option URG \
  --file C:\payloads\pain001.xml
```

## 8. Ergonomie REST

L'API doit egalement permettre:

- une soumission explicite champ par champ;
- une soumission par reference de profil;
- une combinaison `profile + override`.

Exemple:

```json
{
  "profile": "sct-corp-credit-transfer",
  "rule": "ebics-send-default",
  "file": {
    "path": "/payloads/pain001.xml"
  },
  "overrides": {
    "declaredAmount": "1520.45"
  }
}
```

## 9. Benefices

- reduction forte de la verbosite CLI/API;
- baisse du risque d'erreur de saisie;
- meilleure coherence avec le contrat technique connu;
- meilleure exploitabilite;
- meilleure reutilisation entre REST, CLI, `updateconf` et UI.

## 10. Risques a eviter

- enfouir toute la semantique EBICS dans `Rule`;
- creer un profil opaque impossible a surcharger;
- permettre a un profil de contourner le `contract-view`;
- multiplier des profils non documentes et non gouvernes.

## 11. Decision recommandee

Pour EBICS payload:

- oui a une reference vers une configuration predefinie;
- oui au maintien de la saisie explicite champ par champ;
- non a la surcharge de `Rule` comme unique abstraction.

La bonne cible est un objet dedie de type `EbicsPayloadProfile`, articulable
avec `Rule` mais distinct.
