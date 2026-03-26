# Spike D - `Transfer`, `Rule` et operations EBICS non payload

## Statut

Conclu.

Decision proposee: `GO sous reserve`.

## 1. Objet

Ce spike verifie si le modele Gateway centre sur:

- `Transfer`
- `Rule`

peut accueillir EBICS sans deformer le protocole.

Il couvre en particulier le cas des ordres EBICS non payload, c'est-a-dire les
ordres ne devant pas etre interpretes comme des transferts de fichier.

## 2. Constat sur l'existant

Le couplage `Transfer -> Rule` est actuellement fort.

References utiles:

- [transfer.go](/c:/MonProjet/Waarp-Gateway/pkg/model/transfer.go)
- [rule.go](/c:/MonProjet/Waarp-Gateway/pkg/model/rule.go)
- [transfers.go](/c:/MonProjet/Waarp-Gateway/pkg/admin/rest/transfers.go)

Points observes:

- `Transfer.BeforeWrite()` exige un `RuleID`;
- `Rule` encapsule des notions de repertoire local/distant, chemin,
  repertoire temporaire et taches;
- l'API REST des transferts est elle aussi structuree autour de la notion de
  regle.

Conclusion:

- `Transfer` est un objet de transfert de fichier Waarp;
- il n'est pas un bon conteneur universel pour toutes les interactions EBICS.

## 3. Impact sur EBICS

Pour EBICS, il faut distinguer deux familles.

### 3.1 Flux EBICS orientes fichier

Exemples:

- telechargement de reports;
- remise de payloads metier;
- eventuels fichiers de signature transportes comme artefacts.

Ces flux peuvent etre projetes vers:

- `Transfer`
- avec une `Rule` technique Gateway

car ils beneficient effectivement:

- du routage de fichiers;
- des repertoires de travail;
- des taches pre/post/error;
- de l'observabilite transfert.

### 3.2 Operations EBICS non payload

Exemples:

- `INI`, `HIA`, `H3K`
- `HPD`, `HKD`, `HTD`, `HAA`
- `PUB`, `HCA`, `HCS`
- polling RTN / ack techniques
- validations de cycle de vie protocolaire

Ces operations ne doivent pas etre forcees dans `Transfer`, car cela
introduirait:

- des `Rule` artificielles;
- des chemins/fichiers factices;
- une supervision trompeuse;
- un glissement du modele Gateway vers une abstraction fausse.

## 4. Nouvelle notion recommandee

Je recommande d'introduire une notion nouvelle:

- `EbicsOperation`

ou, si on veut rester plus generique a terme:

- `ProtocolOperation`

Pour le perimetre EBICS, la variante la plus claire a court terme est
`EbicsOperation`.

## 5. Proposition de modele minimal

`EbicsOperation` doit porter au minimum:

- identifiant interne;
- type d'operation (`INIT`, `ADMIN`, `REPORTING`, `KEY_MGMT`, `RTN`, `EDS`);
- ordre EBICS (`HPD`, `INI`, `HAC`, etc.);
- `hostId`, `partnerId`, `userId`;
- identifiants de correlation (`transactionId`, `orderId`, `requestId`);
- statut de cycle de vie;
- horodatages;
- resume technique / codes retour;
- pointeurs eventuels vers artefacts techniques ou payloads;
- metadonnees d'integration metier.

## 6. Frontiere avec `Transfer`

La regle de projection devient:

- si l'operation EBICS produit ou consomme un fichier a remettre ou a recevoir,
  on peut creer un `Transfer` associe;
- sinon, seule une `EbicsOperation` est creee.

Cela implique un lien optionnel du type:

- `transfer_id` porte par `EbicsOperation`
- ou une association dediee entre `EbicsOperation` et `Transfer`

L'important est de ne pas imposer le `Transfer`.

Note importante:

- `TransferInfo` peut rester utile comme metadonnee interne ou d'affichage;
- en revanche, il ne doit pas etre considere comme un support
  d'interoperabilite robuste avec un systeme tiers non Waarp.

## 7. Role residuel de `Rule` en EBICS

`Rule` reste utile, mais seulement comme politique technique Waarp.

Elle peut porter:

- le repertoire de depot des reports;
- le repertoire de sortie des payloads remises au metier;
- des taches techniques de copie, renommage, archivage, publication locale;
- des restrictions d'usage Waarp.

Elle ne doit pas porter:

- les permissions EBICS banque;
- les autorisations de signataires;
- le contrat metier;
- la validite d'un ordre EBICS administratif.

## 8. Decision du spike

`GO sous reserve`

Motif:

- Gateway reste un bon candidat si les operations EBICS non payload disposent
  d'un objet dedie;
- en revanche, `NO-GO` si l'implementation essaie de faire rentrer tous les
  ordres EBICS dans `Transfer`.

## 9. Reserves a lever

- choisir `EbicsOperation` ou `ProtocolOperation`;
- decider si l'objet vit dans une table unique ou une famille de tables EBICS;
- definir sa presence dans REST/CLI/UI;
- definir la relation exacte avec `Transfer` pour les cas mixtes.

## 10. Recommendation

Pour avancer vite sans se tromper:

1. creer conceptuellement `EbicsOperation` dans les specs;
2. reserver `Transfer` aux seuls flux fichier;
3. conserver `Rule` comme politique technique optionnelle;
4. prevoir une supervision distincte:
   - vue `Operations EBICS`
   - vue `Transfers`

## 11. Resultat architectural

Le point de friction principal entre Gateway et EBICS est maintenant explicite.

La bonne reponse n'est pas de supprimer `Rule`, ni de la generaliser de force.
La bonne reponse est:

- garder `Rule` la ou elle cree de la valeur;
- ajouter un objet operationnel EBICS pour ce qui n'est pas un transfert.
